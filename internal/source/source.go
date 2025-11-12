package source

type ServiceSource interface {
	CopyTo(destDir string) error
	String() string
}
