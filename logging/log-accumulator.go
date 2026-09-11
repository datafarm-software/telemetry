package logging

import "maps"

type DFLogAccumulator struct {
	m Metadata
}

func (d *DFLogAccumulator) AddMetadata(m Metadata) error {
	if m == nil {
		return nil
	}
	if d.m == nil {
		d.m = m
	} else {
		maps.Copy(d.m, m)
	}
	return nil
}

func (d *DFLogAccumulator) Metadata() Metadata {
	if d.m == nil {
		return Metadata{}
	}
	return d.m
}
