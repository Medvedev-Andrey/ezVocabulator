package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	mWApiToken string
)

const (
	mWRequestFormat   = "https://dictionaryapi.com/api/v3/references/collegiate/json/%s?key=%s"
	mWAudioLinkFormat = "https://media.merriam-webster.com/audio/prons/en/us/mp3/%s/%s.mp3"
)

type mWDictionaryResponse struct {
	Entries []mWEntry
}

type mWEntry struct {
	Meta               mWEntryMeta            `json:"meta"`
	HeadwordInfo       mWHeadwordInfo         `json:"hwi"`
	PartOfSpeech       string                 `json:"fl"`
	Inflections        []mWInflection         `json:"ins"`
	GrammaticalNote    string                 `json:"gram"`
	DefinitionSections []mWDefinitionsSection `json:"def"`
}

type mWInflection struct {
	// Either if or ifc has to be displayed to user
	// if is preferrable
	Inflection        string `json:"if"`
	InflectionCutback string `json:"ifc"`

	// Italic, followed by space
	Label string `json:"il"`
}

type mWEntryMeta struct {
	EntryID string `json:"id"`
}

type mWHeadwordInfo struct {
	Headword       string            `json:"hw"`
	Pronunciations []mWPronunciation `json:"prs"`
}

type mWPronunciation struct {
	Transcription          string  `json:"mw"`
	Audio                  mWAudio `json:"sound"`
	PreLabel               string  `json:"l"`
	PostLabel              string  `json:"l2"`
	RecommendedPunctuation string  `json:"pun"`
}

type mWAudio struct {
	FileName string `json:"audio"`
}

type mWDefinitionsSection struct {
	SubjectLabels mWSenseStatusLabels `json:"sls"`
	SenseSequence mWSensesSequence    `json:"sseq"`
	VerbDivider   string              `json:"vd"`
}

type mWSenseStatusLabels struct {
	Labels []string
}

func (s *mWSenseStatusLabels) UnmarshalJSON(data []byte) error {
	var sections []*json.RawMessage
	err := json.Unmarshal(data, &sections)
	if err != nil {
		return err
	}

	for _, section := range sections {
		var label string
		err = json.Unmarshal(*section, &label)
		if err != nil {
			continue
		}

		s.Labels = append(s.Labels, label)
	}

	return nil
}

type mWSensesSequence struct {
	Items []mWSensesSequenceItem
}

type mWSensesSequenceItem struct {
	BindingSubstitution         *mWBindingSubstitution
	ParenthesizedSenseSequences []mWParenthesizedSenseSequence
	Senses                      []mWSense
}

func (s *mWSensesSequence) UnmarshalJSON(data []byte) error {
	var sequence []*json.RawMessage
	err := json.Unmarshal(data, &sequence)
	if err != nil {
		return err
	}

	for _, section := range sequence {
		var sensesSequenceItem mWSensesSequenceItem

		var subsequence []*json.RawMessage
		err = json.Unmarshal(*section, &subsequence)
		if err != nil {
			return err
		}

		for _, item := range subsequence {
			var itemParts []*json.RawMessage
			err = json.Unmarshal(*item, &itemParts)
			if err != nil {
				return err
			}

			if len(itemParts) != 2 {
				continue
			}

			var sectionName string
			err = json.Unmarshal(*itemParts[0], &sectionName)
			if err != nil {
				return err
			}

			switch {
			case sectionName == "bs":
				var bindingSubstitution mWBindingSubstitution
				err = json.Unmarshal(*itemParts[1], &bindingSubstitution)
				if err == nil {
					sensesSequenceItem.BindingSubstitution = &bindingSubstitution
				}
			case sectionName == "sense":
				var sense mWSense
				err = json.Unmarshal(*itemParts[1], &sense)
				if err == nil {
					sensesSequenceItem.Senses = append(sensesSequenceItem.Senses, sense)
				}
			case sectionName == "pseq":
				var parenthesizedSenseSequence mWParenthesizedSenseSequence
				err = json.Unmarshal(*itemParts[1], &parenthesizedSenseSequence)
				if err == nil {
					sensesSequenceItem.ParenthesizedSenseSequences = append(sensesSequenceItem.ParenthesizedSenseSequences, parenthesizedSenseSequence)
				}
			}

			if err != nil {
				return err
			}
		}

		s.Items = append(s.Items, sensesSequenceItem)
	}

	return nil
}

type mWParenthesizedSenseSequence struct {
	BindingSubstitution *mWBindingSubstitution
	Senses              []mWSense
}

func (s *mWParenthesizedSenseSequence) UnmarshalJSON(data []byte) error {
	var sections []*json.RawMessage
	err := json.Unmarshal(data, &sections)
	if err != nil {
		return err
	}

	for _, section := range sections {
		var subsections []*json.RawMessage
		err = json.Unmarshal(*section, &subsections)
		if err != nil {
			return err
		}

		if len(subsections) != 2 {
			continue
		}

		var subsecionName string
		err = json.Unmarshal(*subsections[0], &subsecionName)
		if err != nil {
			return err
		}

		switch {
		case subsecionName == "sense":
			var sense mWSense
			err = json.Unmarshal(*subsections[1], &sense)
			if err == nil {
				s.Senses = append(s.Senses, sense)
			}
		case subsecionName == "bs":
			var bindingSubstitute mWBindingSubstitution
			err = json.Unmarshal(*subsections[1], &bindingSubstitute)
			if err == nil {
				s.BindingSubstitution = &bindingSubstitute
			}
		}

		if err != nil {
			return err
		}
	}

	return nil
}

type mWBindingSubstitution struct {
	Sense mWSense `json:"sense"`
}

type mWSense struct {
	SenseOrder        string              `json:"sn"`
	DefiningText      mWDefiningText      `json:"dt"`
	SenseStatusLabels mWSenseStatusLabels `json:"sls"`
	DividedSense      *mWDividedSense     `json:"sdsense"`
}

type mWDividedSense struct {
	SenseDivider   string         `json:"sd"`
	DefinitionText mWDefiningText `json:"dt"`
}

type mWDefiningText struct {
	Text       string
	InfoNotes  *mWSupplimentalNote
	UsageNotes []mWUsageNote
	Examples   []mWVerbalIllustration
}

func (s *mWDefiningText) UnmarshalJSON(data []byte) error {
	var contents []*json.RawMessage
	err := json.Unmarshal(data, &contents)
	if err != nil {
		return err
	}

	for _, content := range contents {
		var subcontents []*json.RawMessage
		err := json.Unmarshal(*content, &subcontents)
		if err != nil {
			return err
		}

		if len(subcontents) != 2 {
			return nil
		}

		var subcontentName string
		err = json.Unmarshal(*subcontents[0], &subcontentName)
		if err != nil {
			return err
		}

		switch {
		case subcontentName == "text":
			err = json.Unmarshal(*subcontents[1], &s.Text)
		case subcontentName == "snote":
			err = json.Unmarshal(*subcontents[1], &s.InfoNotes)
		case subcontentName == "uns":
			err = json.Unmarshal(*subcontents[1], &s.UsageNotes)
		case subcontentName == "vis":
			err = json.Unmarshal(*subcontents[1], &s.Examples)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

type mWSupplimentalNote struct {
	Text     string
	Examples []mWVerbalIllustration
}

func (s *mWSupplimentalNote) UnmarshalJSON(data []byte) error {
	var contents []*json.RawMessage
	err := json.Unmarshal(data, &contents)
	if err != nil {
		return err
	}

	for _, content := range contents {
		var subcontents []*json.RawMessage
		err := json.Unmarshal(*content, &subcontents)
		if err != nil {
			return err
		}

		if len(subcontents) != 2 {
			return nil
		}

		var subcontentName string
		err = json.Unmarshal(*subcontents[0], &subcontentName)
		if err != nil {
			return err
		}

		switch {
		case subcontentName == "t":
			err = json.Unmarshal(*subcontents[1], &s.Text)
		case subcontentName == "vis":
			err = json.Unmarshal(*subcontents[1], &s.Examples)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

type mWUsageNote struct {
	Text     string
	Examples []mWVerbalIllustration
}

func (s *mWUsageNote) UnmarshalJSON(data []byte) error {
	var contents []*json.RawMessage
	err := json.Unmarshal(data, &contents)
	if err != nil {
		return err
	}

	for _, content := range contents {
		var subcontents []*json.RawMessage
		err := json.Unmarshal(*content, &subcontents)
		if err != nil {
			return err
		}

		if len(subcontents) != 2 {
			return nil
		}

		var subcontentName string
		err = json.Unmarshal(*subcontents[0], &subcontentName)
		if err != nil {
			return err
		}

		switch {
		case subcontentName == "text":
			err = json.Unmarshal(*subcontents[1], &s.Text)
		case subcontentName == "vis":
			var examples []mWVerbalIllustration
			err = json.Unmarshal(*subcontents[1], &examples)
			if err == nil {
				s.Examples = append(s.Examples, examples...)
			}
		}

		if err != nil {
			return err
		}
	}

	return nil
}

type mWVerbalIllustration struct {
	Text string `json:"t"`
}

func getSubdirectoryForAudio(fileName string) string {
	if len(fileName) == 0 {
		return ""
	}

	if strings.HasPrefix(fileName, "bix") {
		return "bix"
	}

	if strings.HasPrefix(fileName, "gg") {
		return "gg"
	}

	first, _ := utf8.DecodeRuneInString(fileName)
	if unicode.IsPunct(first) || unicode.IsDigit(first) {
		return "number"
	}

	return fileName[0:1]
}

func getDefinitionFromMWDictionary(item string) (*mWDictionaryResponse, error) {
	if mWApiToken == "" {
		mWApiToken = os.Getenv("MW_DICTIONARY_API_TOKEN")
		if mWApiToken == "" {
			log.Fatalf("Environment variable for Merriam Webster Dictionary is not set")
			return nil, fmt.Errorf("Empty Merriam Webster Dictionary API token")
		}
	}

	item = strings.ToLower(item)
	requestUrl := fmt.Sprintf(mWRequestFormat, url.PathEscape(item), mWApiToken)
	request, _ := http.NewRequest("GET", requestUrl, nil)

	contents, err := processRequest(request)
	if err != nil {
		fmt.Printf("Failed getting meanings from Merriam Webster Dictionary for '%s'", item)
		return nil, err
	}

	var response mWDictionaryResponse
	err = json.Unmarshal(contents, &response.Entries)
	if err != nil {
		fmt.Printf("Failed response deserialization from Merriam Webster Dictionary for '%s'", item)
		return nil, err
	}

	return &response, nil
}

func convertMWDictionaryResponse(mWResponse *mWDictionaryResponse) *responseContent {
	var builder responseBuilder

	isFirst := true
	for _, mWEntry := range mWResponse.Entries {
		if isFirst {
			isFirst = false
		} else {
			builder.append("\n")
		}

		builder.append(fmt.Sprintf("▫️%s", mWEntry.HeadwordInfo.Headword))
		if mWEntry.PartOfSpeech != "" {
			builder.append(fmt.Sprintf(" <code>%s</code>\n", mWEntry.PartOfSpeech))
		} else {
			builder.append("\n")
		}

		pronunciations := formatMWPronunciations(mWEntry.HeadwordInfo.Pronunciations)
		if pronunciations != "" {
			builder.append(pronunciations)
			builder.append("\n")
		}

		for _, defenitionSection := range mWEntry.DefinitionSections {
			if defenitionSection.VerbDivider != "" {
				builder.append(fmt.Sprintf("[<i>%s</i>]\n", defenitionSection.VerbDivider))
			}

			for _, senseSection := range defenitionSection.SenseSequence.Items {
				if senseSection.BindingSubstitution != nil {
					builder.append(formatMWSense(senseSection.BindingSubstitution.Sense))
					builder.append("\n")
				}

				for _, parenthesizedSenseSeqense := range senseSection.ParenthesizedSenseSequences {
					if parenthesizedSenseSeqense.BindingSubstitution != nil {
						builder.append(formatMWSense(parenthesizedSenseSeqense.BindingSubstitution.Sense))
						builder.append("\n")
					}

					for _, sense := range parenthesizedSenseSeqense.Senses {
						builder.append(formatMWSense(sense))
						builder.append("\n")
					}
				}

				for _, sense := range senseSection.Senses {
					builder.append(formatMWSense(sense))
					builder.append("\n")
				}
			}
		}
	}

	responseContent := builder.finish()
	responseContent.content = processMWString(responseContent.content)

	return responseContent
}

func formatMWPronunciations(pronunciations []mWPronunciation) string {
	var sb strings.Builder

	isFirst := true
	for _, mWPronunciation := range pronunciations {
		if mWPronunciation.Audio.FileName == "" && mWPronunciation.Transcription == "" {
			continue
		}

		if isFirst {
			sb.WriteRune('\\')
		}

		audioUrl := ""
		if mWPronunciation.Audio.FileName != "" {
			audioUrl = fmt.Sprintf(mWAudioLinkFormat, getSubdirectoryForAudio(mWPronunciation.Audio.FileName), mWPronunciation.Audio.FileName)
		}

		if mWPronunciation.Transcription != "" {
			if audioUrl == "" {
				sb.WriteString(mWPronunciation.Transcription)
			} else {
				sb.WriteString(fmt.Sprintf("<a href=\"%s\">%s 🎧</a>", audioUrl, mWPronunciation.Transcription))
			}
		} else {
			sb.WriteString(fmt.Sprintf("<a href=\"%s\">🎧</a>", audioUrl))
		}

		sb.WriteRune('\\')
		isFirst = false
	}

	return sb.String()
}

func formatMWSense(sense mWSense) string {
	var sb strings.Builder

	if sense.SenseOrder != "" {
		sb.WriteString(fmt.Sprintf("<b>%s</b>", sense.SenseOrder))
	}

	sb.WriteString(formatMWDefiningText(sense.DefiningText))

	if sense.DividedSense != nil {
		definingText := formatMWDefiningText(sense.DividedSense.DefinitionText)
		sb.WriteString(fmt.Sprintf("\n<i>%s</i>%s", sense.DividedSense.SenseDivider, definingText))
	}

	return sb.String()
}

func formatMWDefiningText(definingText mWDefiningText) string {
	var sb strings.Builder
	sb.WriteString(definingText.Text)

	for _, usageNote := range definingText.UsageNotes {
		if usageNote.Text != "" {
			sb.WriteString(fmt.Sprintf("— %s", usageNote.Text))
		}

		for _, usageNoteExample := range usageNote.Examples {
			if usageNoteExample.Text != "" {
				sb.WriteString(fmt.Sprintf("\n// %s", usageNoteExample.Text))
			}
		}
	}

	if definingText.InfoNotes != nil {
		sb.WriteString(fmt.Sprintf("— %s", definingText.InfoNotes.Text))
		for _, example := range definingText.InfoNotes.Examples {
			sb.WriteString(fmt.Sprintf("\n// %s", example.Text))
		}
	}

	for _, example := range definingText.Examples {
		if example.Text != "" {
			sb.WriteString(fmt.Sprintf("\n// %s", example.Text))
		}
	}

	return sb.String()
}

func processMWString(mWString string) string {
	mWString = strings.ReplaceAll(mWString, "{b}", "<b>")        // display text in bold (opening)
	mWString = strings.ReplaceAll(mWString, "{/b}", "</b>")      // display text in bold (closing)
	mWString = strings.ReplaceAll(mWString, "{bc}", "<b>: </b>") // output a bold colon and a space
	mWString = strings.ReplaceAll(mWString, "{it}", "<i>")       // display text in italics (opening)
	mWString = strings.ReplaceAll(mWString, "{/it}", "</i>")     // display text in italics (closing)
	mWString = strings.ReplaceAll(mWString, "{ldquo}", "“")      // output a left double quote character (U+201C: “)
	mWString = strings.ReplaceAll(mWString, "{rdquo}", "”")      // output a right double quote character (U+201D: ”)
	mWString = strings.ReplaceAll(mWString, "{sc}", "")          // display text in small capitals (opening)
	mWString = strings.ReplaceAll(mWString, "{/sc}", "")         // display text in small capitals (closing)
	mWString = strings.ReplaceAll(mWString, "{inf}", " [^")      // display text in subscript (opening)
	mWString = strings.ReplaceAll(mWString, "{/inf}", "] ")      // display text in subscript (closing)
	mWString = strings.ReplaceAll(mWString, "{sup}", " [_")      // display text in superscript (opening)
	mWString = strings.ReplaceAll(mWString, "{/sup}", "] ")      // display text in superscript (closing)

	mWString = strings.ReplaceAll(mWString, "{gloss}", "[")          // encloses a gloss explaining how a word or phrase is used in a particular context (opening)
	mWString = strings.ReplaceAll(mWString, "{/gloss}", "]")         // encloses a gloss explaining how a word or phrase is used in a particular context (closing)
	mWString = strings.ReplaceAll(mWString, "{parahw}", "<b>")       // encloses an instance of the headword within a paragraph label (opening)
	mWString = strings.ReplaceAll(mWString, "{/qword}", "</b>")      // encloses an instance of the headword within a paragraph label (closing)
	mWString = strings.ReplaceAll(mWString, "{phrase}", "<b><i>")    // encloses a phrase in running text (this may be a phrase containing the headword or a defined run-on phrase) (opening)
	mWString = strings.ReplaceAll(mWString, "{/phrase}", "</i></b>") // encloses a phrase in running text (this may be a phrase containing the headword or a defined run-on phrase) (closing)
	mWString = strings.ReplaceAll(mWString, "{qword}", "<i>\"")      // encloses an instance of the headword within a quote (opening)
	mWString = strings.ReplaceAll(mWString, "{/qword}", "\"</i>")    // encloses an instance of the headword within a quote (closing)
	mWString = strings.ReplaceAll(mWString, "{wi}", "<b><i>")        // encloses an instance of the headword used in running text (opening)
	mWString = strings.ReplaceAll(mWString, "{/wi}", "</i></b>")     // encloses an instance of the headword used in running text (closing)

	return mWString
}
