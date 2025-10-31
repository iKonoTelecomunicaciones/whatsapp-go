package whatsappclouddb

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
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
	FROM wb_application
`
const insertAppQuery = `
	INSERT INTO wb_application (name, admin_user, business_id, wb_phone_id, page_access_token)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING *
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
	ctx context.Context, business_id string, phoneID string, name string,
) ([]*CloudRequest, error) {
	whereClauses := ""
	var args = []any{}
	argNum := 1

	if business_id == "" && phoneID == "" {
		zerolog.Ctx(ctx).Error().Msgf("The business_id and phoneID can not be empty")
		return nil, fmt.Errorf("the business_id and phoneID can not be empty")
	}

	if business_id != "" {
		args = append(args, business_id)
		whereClauses += fmt.Sprintf(" WHERE business_id = $%d", argNum)
		argNum++
	}

	if phoneID != "" && business_id == "" {
		args = append(args, phoneID)
		whereClauses += fmt.Sprintf(" WHERE wb_phone_id = $%d", argNum)
		argNum++
	}

	if phoneID != "" && business_id != "" {
		whereClauses += fmt.Sprintf(" AND wb_phone_id = $%d", argNum)
		args = append(args, phoneID)
		argNum++
	}

	if name != "" {
		whereClauses += fmt.Sprintf(" AND name = $%d", argNum)
		args = append(args, name)
		argNum++
	}

	query := getAppByBusinessIDQuery + whereClauses

	apps, err := cloud.QueryMany(ctx, query, args...)
	return apps, err
}

func (cloud *CloudRequestQuery) CreateApp(
	ctx context.Context,
	name string,
	admin_user string,
	business_id string,
	wb_phone_id string,
	page_access_token string,
) (*CloudRequest, error) {
	cloud_insert, err := cloud.QueryOne(ctx, insertAppQuery,
		name, admin_user, business_id, wb_phone_id, page_access_token,
	)

	return cloud_insert, err
}
