package cronx

// ScheduleKind describes what shape of schedule an expression denotes.
type ScheduleKind int

const (
	// KindFields is a five-field schedule, including the named aliases that
	// expand to one
	KindFields ScheduleKind = iota
	// KindInterval repeats on a fixed duration: @every
	KindInterval
	// KindReboot runs once at system startup: @reboot
	KindReboot
)

// String returns the string representation of the schedule kind
func (k ScheduleKind) String() string {
	switch k {
	case KindFields:
		return "fields"
	case KindInterval:
		return "interval"
	case KindReboot:
		return "reboot"
	default:
		return "unknown"
	}
}

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
