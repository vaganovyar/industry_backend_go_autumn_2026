package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)
	n := len(runes)
	if n <= 1 {
		return string(runes)
	}

	k := shift % n
	if k < 0 {
		k += n
	} else if k == 0 {
		return string(runes)
	}

	return string(append(runes[k:], runes[:k]...))
}
