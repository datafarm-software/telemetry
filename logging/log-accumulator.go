package logging

import "maps"

type DFLogAccumulator struct {
	m Metadata
}

func (d *DFLogAccumulator) AddMetadata(m Metadata) {
	if m == nil {
		return
	}
	if d.m == nil {
		d.m = m
	} else {
		maps.Copy(d.m, m)
	}
}

func (d *DFLogAccumulator) Metadata() Metadata {
	if d.m == nil {
		return Metadata{}
	}
	return d.m
}
