package main

const (
	MaxIteration int = 7
)

func trainingIterationToDays(iteration int) int {
	switch iteration {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	case 4:
		return 5
	case 5:
		return 8
	case 6:
		return 13
	default:
		return 21
	}
}
