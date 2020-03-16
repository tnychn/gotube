package gotube

type Error interface {
	Name() string
	Error() string
}
