package code

type Runner interface {
	Run(code string) (string, error)
}
