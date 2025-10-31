package types

type CloudValue struct {
	MessagingProduct string `json:"messaging_product"`
	Metadata         struct {
		DisplayPhoneNumber string `json:"display_phone_number"`
		PhoneNumberID      string `json:"phone_number_id"`
	} `json:"metadata"`
	Contacts []struct {
		Profile struct {
			Name string `json:"name"`
		} `json:"profile"`
		WaID string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		From string `json:"from"`
		ID   string `json:"id"`
		Type string `json:"type"`
		Text struct {
			Body string `json:"body"`
		} `json:"text"`
		TimeStamp string `json:"timestamp"`
	} `json:"messages"`
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
