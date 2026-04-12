package chat

import (
	"io"

	"github.com/cloudwego/eino/adk"
	chatapp "github.com/cy77cc/OpsPilot/internal/modules/ai/chat/app"
	chathttp "github.com/cy77cc/OpsPilot/internal/modules/ai/chat/http"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

type Service = chatapp.Service
type RouteService = chatapp.RouteService
type HTTPHandler = chathttp.HTTPHandler
type SSEWriter = chathttp.SSEWriter

var NewService = chatapp.NewService
var NewServiceWithLogic = chatapp.NewServiceWithLogic
var NewServiceWithLogicAndRouter = chatapp.NewServiceWithLogicAndRouter
var NewServiceWithDB = chatapp.NewServiceWithDB
var NewHTTPHandler = chathttp.NewHTTPHandler
var NewSSEWriter = chathttp.NewSSEWriter

var _ = logic.ChatInput{}
var _ = svc.ServiceContext{}
var _ = gorm.DB{}
var _ io.Writer
var _ adk.ResumableAgent
