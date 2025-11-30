package extractor

// registry holds all registered extractors
var registry []Extractor

// Register adds an extractor to the registry
func Register(e Extractor) {
	registry = append(registry, e)
}

// Match finds the first extractor that can handle the URL
func Match(url string) Extractor {
	for _, e := range registry {
		if e.Match(url) {
			return e
		}
	}
	return nil
}

// List returns all registered extractors
func List() []Extractor {
	return registry
}

func init() {
	// Register extractors in order of priority
	Register(&TwitterExtractor{})
	// Register(&DirectExtractor{})  // TODO: add direct MP4 support
}
