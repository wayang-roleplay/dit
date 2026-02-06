package classifier

import (
	"math"

	"github.com/PuerkitoBio/goquery"
	"github.com/happyhackingspace/dit/internal/vectorizer"
)

// PageTypeModel holds a trained page type classifier.
type PageTypeModel struct {
	Classes   []string             `json:"classes"`
	Coef      [][]float64          `json:"coef"`
	Intercept []float64            `json:"intercept"`
	Pipelines []SerializedPipeline `json:"pipelines"`

	// Runtime state (not serialized)
	dictVecs  []*vectorizer.DictVectorizer
	tfidfVecs []*vectorizer.TfidfVectorizer
	vecTypes  []string
	vecDims   []int
}

// PageTypeTrainConfig holds training configuration for the page type model.
type PageTypeTrainConfig struct {
	C       float64
	MaxIter int
	Verbose bool
}

// DefaultPageTypeTrainConfig returns default training config.
func DefaultPageTypeTrainConfig() PageTypeTrainConfig {
	return PageTypeTrainConfig{
		C:       5.0,
		MaxIter: 100,
	}
}

// Classify returns the predicted page type.
func (m *PageTypeModel) Classify(doc *goquery.Document, formResults []ClassifyResult) string {
	proba := m.ClassifyProba(doc, formResults)
	bestClass := ""
	bestProb := -1.0
	for cls, prob := range proba {
		if prob > bestProb {
			bestProb = prob
			bestClass = cls
		}
	}
	return bestClass
}

// ClassifyProba returns probabilities for each page type.
func (m *PageTypeModel) ClassifyProba(doc *goquery.Document, formResults []ClassifyResult) map[string]float64 {
	features := m.extractFeatures(doc, formResults)

	numClasses := len(m.Classes)
	logits := make([]float64, numClasses)
	for c := range numClasses {
		logits[c] = features.Dot(m.Coef[c]) + m.Intercept[c]
	}

	probs := softmax(logits)
	result := make(map[string]float64, numClasses)
	for c, cls := range m.Classes {
		result[cls] = probs[c]
	}
	return result
}

// extractFeatures runs all page pipelines and concatenates feature vectors.
func (m *PageTypeModel) extractFeatures(doc *goquery.Document, formResults []ClassifyResult) vectorizer.SparseVector {
	pipelines := DefaultPageFeaturePipelines()
	vectors := make([]vectorizer.SparseVector, len(pipelines))

	for i, pipe := range pipelines {
		switch m.vecTypes[i] {
		case "dict":
			feats := pipe.Extractor.ExtractDict(doc, formResults)
			vectors[i] = m.dictVecs[i].Transform(feats)
		case "tfidf":
			text := pipe.Extractor.ExtractString(doc, formResults)
			vectors[i] = m.tfidfVecs[i].Transform(text)
		}
	}

	return vectorizer.ConcatSparse(vectors)
}

// InitRuntime initializes runtime state from serialized pipelines.
func (m *PageTypeModel) InitRuntime() {
	m.dictVecs = make([]*vectorizer.DictVectorizer, len(m.Pipelines))
	m.tfidfVecs = make([]*vectorizer.TfidfVectorizer, len(m.Pipelines))
	m.vecTypes = make([]string, len(m.Pipelines))
	m.vecDims = make([]int, len(m.Pipelines))

	for i, p := range m.Pipelines {
		m.vecTypes[i] = p.VecType
		switch p.VecType {
		case "dict":
			m.dictVecs[i] = p.DictVec
			m.vecDims[i] = p.DictVec.VocabSize()
		case "tfidf":
			m.tfidfVecs[i] = p.TfidfVec
			m.vecDims[i] = p.TfidfVec.VocabSize()
		}
	}
}

// TrainPageType trains a page type classifier.
func TrainPageType(docs []*goquery.Document, formResults [][]ClassifyResult, urls []string, labels []string, config PageTypeTrainConfig) *PageTypeModel {
	pipelines := DefaultPageFeaturePipelines()

	model := &PageTypeModel{}
	model.Pipelines = make([]SerializedPipeline, len(pipelines))
	model.dictVecs = make([]*vectorizer.DictVectorizer, len(pipelines))
	model.tfidfVecs = make([]*vectorizer.TfidfVectorizer, len(pipelines))
	model.vecTypes = make([]string, len(pipelines))
	model.vecDims = make([]int, len(pipelines))

	allVectors := make([][]vectorizer.SparseVector, len(pipelines))

	for i, pipe := range pipelines {
		model.vecTypes[i] = pipe.VecType
		sp := SerializedPipeline{
			Name:          pipe.Name,
			ExtractorType: pageExtractorTypeName(pipe.Extractor),
			VecType:       pipe.VecType,
		}

		// Inject URL into PageURLExtractor
		extractor := pipe.Extractor

		switch pipe.VecType {
		case "dict":
			dv := vectorizer.NewDictVectorizer()
			data := make([]map[string]any, len(docs))
			for j, doc := range docs {
				data[j] = extractor.ExtractDict(doc, formResults[j])
			}
			vecs := dv.FitTransform(data)
			allVectors[i] = vecs
			model.dictVecs[i] = dv
			model.vecDims[i] = dv.VocabSize()
			sp.DictVec = dv

		case "tfidf":
			stopWords := pipe.StopWords
			if pipe.UseEnglishStop {
				stopWords = vectorizer.EnglishStopWords()
			}
			tv := vectorizer.NewTfidfVectorizer(pipe.NgramRange, pipe.MinDF, pipe.Binary, pipe.Analyzer, stopWords)
			corpus := make([]string, len(docs))
			for j, doc := range docs {
				// Handle URL extractor specially
				if _, ok := extractor.(PageURLExtractor); ok {
					corpus[j] = PageURLExtractor{URL: urls[j]}.ExtractString(doc, formResults[j])
				} else {
					corpus[j] = extractor.ExtractString(doc, formResults[j])
				}
			}
			vecs := tv.FitTransform(corpus)
			allVectors[i] = vecs
			model.tfidfVecs[i] = tv
			model.vecDims[i] = tv.VocabSize()
			sp.TfidfVec = tv
		}

		model.Pipelines[i] = sp
	}

	n := len(docs)
	xData := make([]vectorizer.SparseVector, n)
	for j := range n {
		vectors := make([]vectorizer.SparseVector, len(pipelines))
		for i := range pipelines {
			vectors[i] = allVectors[i][j]
		}
		xData[j] = vectorizer.ConcatSparse(vectors)
	}

	classSet := make(map[string]int)
	var classes []string
	for _, l := range labels {
		if _, ok := classSet[l]; !ok {
			classSet[l] = len(classes)
			classes = append(classes, l)
		}
	}
	model.Classes = classes

	totalDim := xData[0].Dim
	numClasses := len(classes)

	y := make([]int, n)
	for j := range n {
		y[j] = classSet[labels[j]]
	}

	reg := config.C
	if reg <= 0 {
		reg = 5.0
	}

	numParams := numClasses * (totalDim + 1)
	params := make([]float64, numParams)

	lbfgs := newLogRegLBFGS(10)
	for iter := range config.MaxIter {
		loss, gradients := logRegObjective(xData, y, params, numClasses, totalDim, reg)

		if config.Verbose && iter%10 == 0 {
			_ = loss
		}

		dir := lbfgs.computeDirection(gradients, numParams)
		step := logRegLineSearch(xData, y, params, dir, numClasses, totalDim, reg, loss)

		prevParams := make([]float64, numParams)
		copy(prevParams, params)
		for i := range numParams {
			params[i] += step * dir[i]
		}

		_, newGrad := logRegObjective(xData, y, params, numClasses, totalDim, reg)
		s := make([]float64, numParams)
		yVec := make([]float64, numParams)
		for i := range numParams {
			s[i] = params[i] - prevParams[i]
			yVec[i] = newGrad[i] - gradients[i]
		}
		lbfgs.update(s, yVec)

		maxGrad := 0.0
		for _, g := range newGrad {
			if math.Abs(g) > maxGrad {
				maxGrad = math.Abs(g)
			}
		}
		if maxGrad < 1e-5 {
			break
		}
	}

	model.Coef = make([][]float64, numClasses)
	model.Intercept = make([]float64, numClasses)
	for c := range numClasses {
		model.Coef[c] = make([]float64, totalDim)
		offset := c * (totalDim + 1)
		copy(model.Coef[c], params[offset:offset+totalDim])
		model.Intercept[c] = params[offset+totalDim]
	}

	return model
}

func pageExtractorTypeName(e PageFeatureExtractor) string {
	switch e.(type) {
	case PageStructureExtractor:
		return "PageStructure"
	case PageTitleExtractor:
		return "PageTitle"
	case PageMetaDescriptionExtractor:
		return "PageMetaDescription"
	case PageHeadingsExtractor:
		return "PageHeadings"
	case PageH1Extractor:
		return "PageH1"
	case PageCSSExtractor:
		return "PageCSS"
	case PageNavTextExtractor:
		return "PageNavText"
	case FormTypeSummaryExtractor:
		return "FormTypeSummary"
	case PageURLExtractor:
		return "PageURL"
	default:
		return "unknown"
	}
}
