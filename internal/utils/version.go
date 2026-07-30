package utils

const DevelVersionIdent = "devel"

func IsDevelVersion(ver string) bool {
	return (ver == "devel" || ver == "localbuilded")
}
