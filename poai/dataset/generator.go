package dataset

import (
	"fmt"
	"math/rand"
)

// ProceduralQuiz generates deterministic quizzes based on block height and nonce
// This ensures each nonce produces unique, verifiable input to the LLM
func ProceduralQuiz(blockHeight uint64, nonce uint64) []string {
	// Create a deterministic seed from block height and nonce
	seed := int64(blockHeight) + int64(nonce)
	rng := rand.New(rand.NewSource(seed))

	// Generate 3-5 quiz questions per block
	numQuestions := 3 + rng.Intn(3) // 3-5 questions
	quizzes := make([]string, numQuestions)

	for i := 0; i < numQuestions; i++ {
		// Mix the nonce and question index for more variability
		questionSeed := seed + int64(i*1000) + int64(nonce%10000)
		qRng := rand.New(rand.NewSource(questionSeed))

		// Generate different types of questions
		questionType := qRng.Intn(4)

		switch questionType {
		case 0: // Math addition
			x := 1 + qRng.Intn(1000)
			y := 1 + qRng.Intn(1000)
			quizzes[i] = fmt.Sprintf("What is %d + %d?", x, y)
		case 1: // Math multiplication
			x := 1 + qRng.Intn(50)
			y := 1 + qRng.Intn(50)
			quizzes[i] = fmt.Sprintf("What is %d × %d?", x, y)
		case 2: // Pattern completion
			start := 1 + qRng.Intn(10)
			step := 1 + qRng.Intn(5)
			quizzes[i] = fmt.Sprintf("Complete the pattern: %d, %d, %d, ?", start, start+step, start+2*step)
		case 3: // Logic puzzle
			items := []string{"apple", "banana", "cherry", "date", "elderberry"}
			idx := qRng.Intn(len(items))
			quizzes[i] = fmt.Sprintf("What fruit comes after %s in alphabetical order?", items[idx])
		}
	}

	return quizzes
}
