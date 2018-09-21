package cwl

type CWLVersion string

func NewCWLVersion(version string) CWLVersion {
	return CWLVersion(version)
}
