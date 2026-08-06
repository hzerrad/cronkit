package cronx

type FieldKind int

const (
	KindEvery FieldKind = iota
	KindStep
	KindBoundedStep
	KindSingle
	KindRange
	KindList
)

func (f *field) Kind() FieldKind {
	if len(f.parts) > 1 {
		return KindList
	}

	p := f.parts[0]
	stepped := p.step > 1

	switch {
	case p.isEvery && stepped:
		return KindStep
	case p.isEvery:
		return KindEvery
	case p.isRange && stepped:
		return KindBoundedStep
	case p.isRange:
		return KindRange
	default:
		return KindSingle
	}
}
