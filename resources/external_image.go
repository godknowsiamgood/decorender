package resources

type ExternalImage interface {
	Prefetch(path string)
	Get(path string) ([]byte, error)
}
