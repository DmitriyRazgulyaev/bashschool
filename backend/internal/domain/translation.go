package domain

type AITranslation struct {
	Correct          string `json:"correct"`
	Incorrect        string `json:"incorrect"`
	ErrorExplanation string `json:"errorExplanation"`
}
