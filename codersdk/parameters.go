package codersdk

import (
	"context"
	"fmt"

	"github.com/coder/coder/v2/codersdk/wsjson"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

func (c *Client) TemplateVersionDynamicParameters(ctx context.Context, userID, version uuid.UUID) (*wsjson.Stream[DynamicParametersResponse, DynamicParametersRequest], error) {
	conn, err := c.Dial(ctx, fmt.Sprintf("/api/v2/users/%s/templateversions/%s/dynamic-parameters", userID, version), nil)
	if err != nil {
		return nil, err
	}
	return wsjson.NewStream[DynamicParametersResponse, DynamicParametersRequest](conn, websocket.MessageText, websocket.MessageText, c.Logger()), nil
}
