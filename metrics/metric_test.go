// Copyright 2011 Google Inc. All Rights Reserved.
// This file is available under the Apache license.

package metrics

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func BenchmarkIncrementScalar(b *testing.B) {
	d := &Datum{}
	ts := time.Now()
	for i := 0; i < b.N; i++ {
		d.IncBy(1, ts)
	}
}

func TestScalarMetric(t *testing.T) {
	v := NewMetric("test", "prog", Counter)
	d, _ := v.GetDatum()
	d.IncBy(1, time.Now())
	lv := v.findLabelValueOrNil([]string{})
	if lv == nil {
		t.Errorf("couldn't find labelvalue")
	}
	newD := lv.Value
	if newD == nil {
		t.Errorf("new_d is nil")
	}
	if newD.Value != 1 {
		t.Errorf("value not 1")
	}
	// TODO: try setting datum with labels on scalar
}

func TestDimensionedMetric(t *testing.T) {
	v := NewMetric("test", "prog", Counter, "foo")
	d, _ := v.GetDatum("a")
	d.IncBy(1, time.Now())
	if v.findLabelValueOrNil([]string{"a"}).Value.Value != 1 {
		t.Errorf("fail")
	}

	v = NewMetric("test", "prog", Counter, "foo", "bar")
	d, _ = v.GetDatum("a", "b")
	d.IncBy(1, time.Now())
	if v.findLabelValueOrNil([]string{"a", "b"}).Value.Value != 1 {
		t.Errorf("fail")
	}

	v = NewMetric("test", "prog", Counter, "foo", "bar", "quux")
	d, _ = v.GetDatum("a", "b", "c")
	d.IncBy(1, time.Now())
	if v.findLabelValueOrNil([]string{"a", "b", "c"}).Value.Value != 1 {
		t.Errorf("fail")
	}
}

var labelSetTests = []struct {
	values         []string
	expectedLabels map[string]string
}{
	{
		[]string{"a", "b", "c"},
		map[string]string{"foo": "a", "bar": "b", "quux": "c"},
	},
	{
		[]string{"a", "b", "d"},
		map[string]string{"foo": "a", "bar": "b", "quux": "d"},
	},
}

func TestEmitLabelSet(t *testing.T) {
	m := NewMetric("test", "prog", Gauge, "foo", "bar", "quux")
	c := make(chan *LabelSet)

	ts := time.Now()

	var expectedLabels []map[string]string
	for _, tc := range labelSetTests {
		d, _ := m.GetDatum(tc.values...)
		d.Set(37, ts)
		expectedLabels = append(expectedLabels, tc.expectedLabels)
	}

	go m.EmitLabelSets(c)

	var labels []map[string]string
	for ls := range c {
		labels = append(labels, ls.Labels)
	}

	// Equivalence for slices is not defined under ==, and DeepEqual does an
	// elementwise comparison.  We can't guarantee that the labels are in
	// order, so do the N^2 comparision.
	if len(labels) != len(expectedLabels) {
		t.Errorf("Label length doesn't match\n\texpected %v\n\treceived %v\n", expectedLabels, labels)
	}

Loop:
	for i := range expectedLabels {
		for j := range labels {
			if reflect.DeepEqual(expectedLabels[i], labels[j]) {
				continue Loop
			}
		}
		t.Errorf("Labels don't match: couldn't find %v in labels\n\texpected %v\n\treceived %v\n", expectedLabels[i], expectedLabels, labels)

	}
}

func TestFindLabelValueOrNil(t *testing.T) {
	m0 := NewMetric("foo", "prog", Counter)
	if r0 := m0.findLabelValueOrNil([]string{}); r0 != nil {
		t.Errorf("m0 should be nil: %v", r0)
	}
	d, err := m0.GetDatum()
	if err != nil {
		t.Errorf("Bad datum %v: %v\n", d, err)
	}
	if r1 := m0.findLabelValueOrNil([]string{}); r1 == nil {
		t.Errorf("m0 should not be nil: %v", r1)
	}
	m1 := NewMetric("bar", "prog", Counter, "a")
	d1, err1 := m1.GetDatum("1")
	if err1 != nil {
		t.Errorf("err1 %v: %v\n", d1, err1)
	}
	if r2 := m1.findLabelValueOrNil([]string{"0"}); r2 != nil {
		t.Errorf("r2 should be nil")
	}
	if r3 := m1.findLabelValueOrNil([]string{"1"}); r3 == nil {
		t.Errorf("r3 should be non nil")
	}
}

func TestMetricJSONRoundTrip(t *testing.T) {
	m := NewMetric("test", "prog", Gauge, "foo", "bar", "quux")
	d, _ := m.GetDatum("a", "2", "d")
	d.Set(1, time.Now())

	j, e := json.Marshal(m)
	if e != nil {
		t.Errorf("json.Marshal failed: %s\n", e)
	}

	r := &Metric{}
	e = json.Unmarshal(j, &r)
	if e != nil {
		t.Errorf("json.Unmarshal failed: %s\n", e)
	}

	if !reflect.DeepEqual(m, r) {
		t.Errorf("Round trip wasn't stable.\n\texpected: %v\n\treceived: %v\n", m, r)
	}
}