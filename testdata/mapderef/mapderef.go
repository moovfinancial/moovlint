package mapderef

type Projection struct {
	ID   string
	Name string
}

type Worker interface{ Do() }

func badField(projections map[string]*Projection) string {
	return projections["abc"].ID // want "map lookup is dereferenced without a comma-ok check"
}

func badInterface(workers map[string]Worker) {
	workers["x"].Do() // want "map lookup is dereferenced without a comma-ok check"
}

func badStar(projections map[string]*Projection) string {
	p := *projections["abc"] // want "map lookup is dereferenced without a comma-ok check"
	return p.ID
}

func goodCommaOK(projections map[string]*Projection) string {
	p, ok := projections["abc"]
	if !ok || p == nil {
		return ""
	}
	return p.ID
}

func goodStructValue(projections map[string]Projection) string {
	return projections["abc"].ID
}
