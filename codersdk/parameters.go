package codersdk

import (
	"context"
	"fmt"

	"github.com/coder/coder/v2/codersdk/wsjson"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

func (c *Client) TemplateVersionDynamicParameters(ctx context.Context, version uuid.UUID) (*wsjson.Stream[DynamicParametersResponse, DynamicParametersRequest], error) {
	conn, err := c.Dial(ctx, fmt.Sprintf("/api/v2/templateversions/%s/dynamic-parameters", version), nil)
	if err != nil {
		return nil, err
	}
	return wsjson.NewStream[DynamicParametersResponse, DynamicParametersRequest](conn, websocket.MessageText, websocket.MessageText, c.Logger()), nil
}
