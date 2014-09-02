package supervisor

type SupervisorCode string

const (
	UNKNOWN_METHOD        SupervisorCode = "1"
	INCORRECT_PARAMETERS  SupervisorCode = "2"
	BAD_ARGUMENTS         SupervisorCode = "3"
	SIGNATURE_UNSUPPORTED SupervisorCode = "4"
	SHUTDOWN_STATE        SupervisorCode = "6"
	BAD_NAME              SupervisorCode = "10"
	NO_FILE               SupervisorCode = "20"
	NOT_EXECUTABLE        SupervisorCode = "21"
	FAILED                SupervisorCode = "30"
	ABNORMAL_TERMINATION  SupervisorCode = "40"
	SPAWN_ERROR           SupervisorCode = "50"
	ALREADY_STARTED       SupervisorCode = "60"
	NOT_RUNNING           SupervisorCode = "70"
	SUCCESS               SupervisorCode = "80"
	ALREADY_ADDED         SupervisorCode = "90"
	STILL_RUNNING         SupervisorCode = "91"
	CANT_REREAD           SupervisorCode = "92"
)

type SupervisorState int

const (
	STOPPED  SupervisorState = 0
	STARTING SupervisorState = 10
	RUNNING  SupervisorState = 20
	BACKOFF  SupervisorState = 30
	STOPPING SupervisorState = 40
	EXITED   SupervisorState = 100
	FATAL    SupervisorState = 200
	UNKNOWN  SupervisorState = 1000
)
