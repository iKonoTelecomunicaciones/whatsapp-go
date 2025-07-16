package whatsappclouddb

import (
	"context"
	"fmt"

	"go.mau.fi/util/dbutil"
)

type CloudRequestQuery struct {
	*dbutil.QueryHelper[*CloudRequest]
}

type CloudRequest struct {
	BusinessID      string `db:"business_id"`
	WbPhoneID       string `db:"app_id"`
	Name            string `db:"name"`
	AdminUser       string `db:"admin_user"`
	PageAccessToken string `db:"page_access_token"`
}

const getAppByBusinessIDQuery = `
	SELECT *
	FROM whatsapp_app
`

func (cloud *CloudRequest) Scan(row dbutil.Scannable) (*CloudRequest, error) {
	err := row.Scan(
		&cloud.BusinessID,
		&cloud.WbPhoneID,
		&cloud.Name,
		&cloud.AdminUser,
		&cloud.PageAccessToken,
	)
	if err != nil {
		return nil, err
	}
	return cloud, nil
}

func (cloud *CloudRequestQuery) SearchApp(
	ctx context.Context, appID string, phoneID string, name string,
) ([]*CloudRequest, error) {
	whereClauses := ""
	args := []any{}
	argNum := 1
	if appID != "" {
		whereClauses += fmt.Sprintf(" AND app_id = $%d", argNum)
		args = append(args, appID)
		argNum++
	}
	if phoneID != "" {
		whereClauses += fmt.Sprintf(" AND phone_id = $%d", argNum)
		args = append(args, phoneID)
		argNum++
	}
	if name != "" {
		whereClauses += fmt.Sprintf(" AND name = $%d", argNum)
		args = append(args, name)
		argNum++
	}

	query := fmt.Sprintf(getAppByBusinessIDQuery, whereClauses)

	apps, err := cloud.QueryMany(ctx, query, args...)
	return apps, err
}
