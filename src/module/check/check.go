package check

// Result holds the outcome of a single dependency check.
type Result struct {
	Name    string
	OK      bool
	Message string
}

// Checker is implemented by any module that can verify its own dependency.
type Checker interface {
	CheckDependency() Result
}
