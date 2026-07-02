package distiller

import "strings"

func ChunkTranscript(filtered string, maxContext int) []string {
	budget := maxContext * 3
	if budget <= 0 {
		budget = 120000
	}
	if len(filtered) <= budget {
		return []string{filtered}
	}
	var chunks []string
	var current strings.Builder
	for _, line := range strings.Split(filtered, "\n") {
		if current.Len()+len(line)+1 > budget && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}
