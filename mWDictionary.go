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
	MWTrancription         string  `json:"mw"`
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
	BindingSubstitution         mWBindingSubstitution
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
					sensesSequenceItem.BindingSubstitution = bindingSubstitution
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
	BindingSubstitute mWBindingSubstitution
	Senses            []mWSense
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
				s.BindingSubstitute = bindingSubstitute
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
	DividedSense      mWDividedSense      `json:"sdsense"`
}

type mWDividedSense struct {
	SenseDivider   string         `json:"sd"`
	DefinitionText mWDefiningText `json:"dt"`
}

type mWDefiningText struct {
	Text       string
	InfoNotes  mWSupplimentalNote
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

func getDefinitionFromMWDictionary(item string) ([]mWEntry, error) {
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

	var entries []mWEntry
	err = json.Unmarshal(contents, &entries)
	if err != nil {
		fmt.Printf("Failed response deserialization from Merriam Webster Dictionary for '%s'", item)
		return nil, err
	}

	return nil, nil
}
