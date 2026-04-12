package todo

// SessionKeyOpsTodos stores the ops todo snapshot in the ADK session.
const SessionKeyOpsTodos = "opspilot_session_key_ops_todos"

// OpsTODO is the persisted ops todo item shape.
type OpsTODO struct {
	Content    string `json:"content"`
	ActiveForm string `json:"active_form"`
	Status     string `json:"status" jsonschema:"enum=pending,enum=in_progress,enum=completed"`
}

type writeOpsTodosArguments struct {
	Todos []OpsTODO `json:"todos"`
}
