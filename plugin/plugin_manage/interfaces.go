package pluginmanage

type VisibilityCustomed interface {
	Visible() bool
}

type Managable interface {
	Managable() bool
}
