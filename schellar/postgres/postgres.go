package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/frinx/schellar/ifc"

	"github.com/jackc/pgx/v4"
	"github.com/sirupsen/logrus"
)

type PostgresDB struct {
	// TODO: pool
	connection pgx.Conn
}

func InitDB() PostgresDB {
	var err error
	connection, err := pgx.Connect(context.Background(), os.Getenv("POSTGRES_DATABASE_URL"))
	if err != nil {
		logrus.Fatalf("Unable to connection to database: %v", err)
	}
	return PostgresDB{*connection}
}

func (db PostgresDB) queryAll(sql string, args ...interface{}) ([]ifc.Schedule, error) {
	rows, err := db.connection.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := make([]ifc.Schedule, 0, len(rows.FieldDescriptions()))
	for rows.Next() {
		var (
			ScheduleName        string
			Enabled             bool
			Status              string
			WorkflowName        string
			WorkflowVersion     int
			WorkflowContext     map[string]interface{}
			CronString          string
			ParallelRuns        bool
			CheckWarningSeconds int
			FromDate            *time.Time
			ToDate              *time.Time
			CorrelationID       string
			TaskToDomain        map[string]string
			LastUpdate          time.Time
		)

		err = rows.Scan(&ScheduleName, &Enabled, &Status, &WorkflowName, &WorkflowVersion,
			&WorkflowContext, &CronString, &ParallelRuns, &CheckWarningSeconds,
			&FromDate, &ToDate, &CorrelationID, &TaskToDomain, &LastUpdate,
		)
		if err != nil {
			return nil, err
		}
		schedule := ifc.Schedule{ScheduleName, Enabled, Status, WorkflowName, WorkflowVersion,
			WorkflowContext, CronString, ParallelRuns, CheckWarningSeconds,
			FromDate, ToDate, LastUpdate, CorrelationID, TaskToDomain}

		schedules = append(schedules, schedule)
	}
	return schedules, nil

}

const rowNames = `
schedule_name,
is_enabled,
workflow_status,
workflow_name,
workflow_version,
workflow_context,
cron_string,
parallel_runs,
check_warning_seconds,
from_date,
to_date,
correlation_id,
task_to_domain,
last_update`

func (db PostgresDB) FindAll() ([]ifc.Schedule, error) {
	return db.queryAll("SELECT " + rowNames + " FROM schedule")
}

func (db PostgresDB) FindAllByEnabled(enabled bool) ([]ifc.Schedule, error) {
	return db.queryAll("SELECT "+rowNames+" FROM schedule WHERE is_enabled=$1", enabled)
}

func (db PostgresDB) FindByName(scheduleName string) (*ifc.Schedule, error) {
	schedules, err := db.queryAll("SELECT "+rowNames+" FROM schedule WHERE name=$1",
		scheduleName)
	if err != nil {
		return nil, err
	}
	if len(schedules) == 1 {
		return &schedules[0], nil
	}
	return nil, errors.New(
		fmt.Sprintf(
			"Unexpected result for FindByName('%s'): Found %d items",
			scheduleName, len(schedules),
		))
}

func (db PostgresDB) FindByStatus(status string) ([]ifc.Schedule, error) {
	return db.queryAll("SELECT "+rowNames+" FROM schedule WHERE workflow_status=$1", status)
}

func (db PostgresDB) Insert(schedule ifc.Schedule) error {
	_, err := db.connection.Exec(context.Background(),
		"INSERT INTO schedule("+rowNames+") VALUES "+sqlParamsRange(14),
		schedule.Name,
		schedule.Enabled,
		schedule.Status,
		schedule.WorkflowName,
		schedule.WorkflowVersion,
		schedule.WorkflowContext,
		schedule.CronString,
		schedule.ParallelRuns,
		schedule.CheckWarningSeconds,
		schedule.FromDate,
		schedule.ToDate,
		schedule.CorrelationID,
		schedule.TaskToDomain,
		schedule.LastUpdate,
	)
	return err
}

func (db PostgresDB) UpdateStatus(scheduleName string, scheduleStatus string) error {
	_, err := db.connection.Exec(context.Background(),
		"UPDATE schedule SET workflow_status=$2 WHERE schedule_name=$1",
		scheduleName, scheduleStatus)
	return err
}

func (db PostgresDB) UpdateStatusAndWorkflowContext(schedule ifc.Schedule) error {
	_, err := db.connection.Exec(context.Background(),
		"UPDATE schedule SET workflow_status=$2, workflow_context=$3 WHERE schedule_name=$1",
		schedule.Name, schedule.Status, schedule.WorkflowContext)
	return err
}

func (db PostgresDB) Update(schedule ifc.Schedule) error {
	_, err := db.connection.Exec(context.Background(),
		`UPDATE schedule SET
			is_enabled=$2,
			workflow_status=$3,
			workflow_name=$4,
			workflow_version=$5,
			workflow_context=$6,
			cron_string=$7,
			parallel_runs=$8,
			check_warning_seconds=$9,
			from_date=$10,
			to_date=$11,
			correlation_id=$12,
			task_to_domain=$13,
			last_update=$14
			WHERE schedule_name=$1`,
		schedule.Name,
		schedule.Enabled,
		schedule.Status,
		schedule.WorkflowName,
		schedule.WorkflowVersion,
		schedule.WorkflowContext,
		schedule.CronString,
		schedule.ParallelRuns,
		schedule.CheckWarningSeconds,
		schedule.FromDate,
		schedule.ToDate,
		schedule.CorrelationID,
		schedule.TaskToDomain,
		schedule.LastUpdate,
	)
	return err
}

func (db PostgresDB) RemoveByName(scheduleName string) error {
	_, err := db.connection.Exec(context.Background(),
		"DELETE FROM schedule WHERE schedule_name=$1", scheduleName)
	return err
}

// Creates string with sql parameters.
// Example: sqlParamsRange(3) returns "($1,$2,$3)"
func sqlParamsRange(max uint) string {
	result := make([]string, 0, max)
	var i uint
	for i = 1; i <= max; i++ {
		result = append(result, fmt.Sprintf("$%d", i))
	}
	return "(" + strings.Join(result, ",") + ")"
}