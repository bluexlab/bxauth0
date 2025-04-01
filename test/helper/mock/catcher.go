package mock

func NewCatcher() *Catcher {
	return &Catcher{}
}

type Catcher struct {
	v any
}

func (m *Catcher) Matches(x any) bool {
	m.v = x
	return true
}

func (m *Catcher) String() string {
	return "is anything"
}

func (m *Catcher) Value() any {
	return m.v
}
