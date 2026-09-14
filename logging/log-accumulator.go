package logging

import "maps"

type dFLogAccumulator struct {
	m Metadata
}

func (d *dFLogAccumulator) AddMetadata(m Metadata) {
	if m == nil {
		return
	}
	if d.m == nil {
		d.m = m
	} else {
		maps.Copy(d.m, m)
	}
}

func (d *dFLogAccumulator) Metadata() Metadata {
	if d.m == nil {
		return Metadata{}
	}
	return d.m
}
