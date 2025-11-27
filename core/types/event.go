package types

type CloudMetaData struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type CloudContact struct {
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
	WaID string `json:"wa_id"`
}

type ImageCloud struct {
	ID       string  `json:"id"`
	MimeType string  `json:"mime_type"`
	SHA256   string  `json:"sha256"`
	Caption  *string `json:"caption"`
}

type VideoCloud struct {
	ID       string  `json:"id"`
	MimeType string  `json:"mime_type"`
	SHA256   string  `json:"sha256"`
	Caption  *string `json:"caption"`
}

type AudioCloud struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Voice    bool   `json:"voice"`
}

type DocumentCloud struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	FileName string `json:"filename"`
}

type StickerCloud struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Animated bool   `json:"animated"`
}

type CloudMessage struct {
	From string `json:"from"`
	ID   string `json:"id"`
	Type string `json:"type"`
	Text *struct {
		Body string `json:"body"`
	} `json:"text"`
	Image     *ImageCloud    `json:"image"`
	Video     *VideoCloud    `json:"video"`
	Audio     *AudioCloud    `json:"audio"`
	Document  *DocumentCloud `json:"document"`
	Sticker   *StickerCloud  `json:"sticker"`
	TimeStamp string         `json:"timestamp"`
	Context   *struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"context"`
}

type CloudErrors []struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data"`
}

type CloudStatuses []struct {
	ID          string      `json:"id"`
	Status      string      `json:"status"`
	Timestamp   string      `json:"timestamp"`
	RecipientID string      `json:"recipient_id"`
	Errors      CloudErrors `json:"errors"`
}

type CloudValue struct {
	MessagingProduct string         `json:"messaging_product"`
	Metadata         CloudMetaData  `json:"metadata"`
	Contacts         []CloudContact `json:"contacts"`
	Messages         []CloudMessage `json:"messages"`
	Statuses         *CloudStatuses `json:"statuses"`
}

type CloudEvent struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value CloudValue `json:"value"`
			Field string     `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

type CloudRegisterAppRequest struct {
	AppName     string  `json:"app_name"`
	WabaID      string  `json:"waba_id"`
	AppPhoneID  string  `json:"app_phone_id"`
	AccessToken string  `json:"access_token"`
	NoticeRoom  string  `json:"notice_room"`
	AdminUser   *string `json:"admin_user"`
}

type CloudUserMetadata struct {
	WabaID          string `json:"waba_id"`
	BusinessPhoneID string `json:"business_phone_id"`
	PageAccessToken string `json:"page_access_token"`
}

type ContactsResponse []struct {
	Input string `json:"input"`
	WaID  string `json:"wa_id"`
}

type MessagesResponse []struct {
	ID string `json:"id"`
}

type CloudMessageResponse struct {
	Contacts         ContactsResponse `json:"contacts"`
	Messages         MessagesResponse `json:"messages"`
	MessagingProduct string           `json:"messaging_product"`
}

type CloudMediaResponse struct {
	ID               string `json:"id"`
	MessagingProduct string `json:"messaging_product"`
	URL              string `json:"url"`
	MimeType         string `json:"mime_type"`
	Hash             string `json:"hash"`
	FileSize         int    `json:"file_size"`
}

type FileInfo struct {
	Size     int
	MimeType string
}

type MediaResponse struct {
	FileName *string
	Url      string
	FileInfo *FileInfo
	Caption  *string
}
