package lang

type Entry struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var Supported = []Entry{
	{Code: "ar", Name: "Arabic"},
	{Code: "bg", Name: "Bulgarian"},
	{Code: "cn", Name: "Simplified Chinese"},
	{Code: "de", Name: "German"},
	{Code: "en", Name: "English"},
	{Code: "es", Name: "Spanish"},
	{Code: "fa", Name: "Farsi"},
	{Code: "fr", Name: "French"},
	{Code: "id", Name: "Indonesian"},
	{Code: "in", Name: "Hindi"},
	{Code: "it", Name: "Italian"},
	{Code: "jp", Name: "Japanese"},
	{Code: "kr", Name: "Korean"},
	{Code: "lv", Name: "Latvian"},
	{Code: "my", Name: "Malay"},
	{Code: "nl", Name: "Dutch"},
	{Code: "pl", Name: "Polish"},
	{Code: "pt", Name: "Portuguese"},
	{Code: "ro", Name: "Romanian"},
	{Code: "ru", Name: "Russian"},
	{Code: "th", Name: "Thai"},
	{Code: "tw", Name: "Traditional Chinese"},
	{Code: "vn", Name: "Vietnamese"},
}

func CodeToName(code string) string {
	for _, l := range Supported {
		if l.Code == code {
			return l.Name
		}
	}
	return code
}

func NameToCode(name string) string {
	for _, l := range Supported {
		if l.Name == name {
			return l.Code
		}
	}
	return name
}
