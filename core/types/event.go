package types

type CloudMetaData struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type CloudContacts []struct {
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
	WaID string `json:"wa_id"`
}

type CloudMessages []struct {
	From string `json:"from"`
	ID   string `json:"id"`
	Type string `json:"type"`
	Text struct {
		Body string `json:"body"`
	} `json:"text"`
	TimeStamp string `json:"timestamp"`
	Context   struct {
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
	MessagingProduct string        `json:"messaging_product"`
	Metadata         CloudMetaData `json:"metadata"`
	Contacts         CloudContacts `json:"contacts"`
	Messages         CloudMessages `json:"messages"`
	Statuses         CloudStatuses `json:"statuses"`
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
