package cmd

// maskToken shows first 4 and last 4 characters with asterisks in between.
func maskToken(tok string) string {
	if len(tok) <= 8 {
		return "********"
	}
	stars := make([]byte, len(tok)-8)
	for i := range stars {
		stars[i] = '*'
	}
	return tok[:4] + string(stars) + tok[len(tok)-4:]
}
