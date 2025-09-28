package dbrepo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

// ============================== Attendance Repository ==============================
type AttendanceRepo struct {
	db *pgxpool.Pool
}

func NewAttendanceRepo(db *pgxpool.Pool) *AttendanceRepo {
	return &AttendanceRepo{db: db}
}

// ----------------- SINGLE UPDATE -----------------

func (a *AttendanceRepo) UpdateTodayAttendance(ctx context.Context, employeeAttendance models.Attendance) error {
	// Insert or update in DB
	query := `
		INSERT INTO attendance (employee_id, work_date, status, check_in, check_out, overtime_hours)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (employee_id, work_date)
		DO UPDATE SET status = EXCLUDED.status,
					  check_in = EXCLUDED.check_in,
					  check_out = EXCLUDED.check_out,
					  overtime_hours = EXCLUDED.overtime_hours,
					  updated_at = CURRENT_TIMESTAMP;
	`

	_, err := a.db.Exec(ctx, query,
		employeeAttendance.EmployeeID,
		employeeAttendance.WorkDate,
		employeeAttendance.Status,
		employeeAttendance.CheckIn,
		employeeAttendance.CheckOut,
		employeeAttendance.OvertimeHours,
	)

	return err
}

// ----------------- BATCH UPDATE -----------------
func (a *AttendanceRepo) BatchUpdateTodayAttendance(ctx context.Context, entries []*models.Attendance) error {
	if len(entries) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, e := range entries {
		batch.Queue(`
			INSERT INTO attendance (employee_id, work_date, status, check_in, check_out, overtime_hours)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (employee_id, work_date)
			DO UPDATE SET status = EXCLUDED.status,
						  check_in = EXCLUDED.check_in,
						  check_out = EXCLUDED.check_out,
						  overtime_hours = EXCLUDED.overtime_hours,
						  updated_at = CURRENT_TIMESTAMP;
		`, e.EmployeeID, e.WorkDate, e.Status, utils.NullableTime(e.CheckIn), utils.NullableTime(e.CheckOut), e.OvertimeHours)
	}

	br := a.db.SendBatch(ctx, batch)
	defer br.Close()

	for _, e := range entries {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to update attendance for employee %d: %w", e.EmployeeID, err)
		}
	}

	return nil
}

// ----------------- CALENDAR -----------------

func (a *AttendanceRepo) GetEmployeeCalendar(ctx context.Context, employeeIDStr, month, start, end string) (*models.EmployeeCalendar, error) {
	// Convert employeeID to int
	empID, err := strconv.Atoi(employeeIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid employee ID: %s", employeeIDStr)
	}

	var query string
	var rows pgx.Rows

	// Month query
	if month != "" {
		monthTime, err := time.Parse("2006-01", month)
		if err != nil {
			return nil, fmt.Errorf("invalid month format, expected YYYY-MM")
		}

		query = `
			SELECT a.id, a.employee_id, e.fname || ' ' || e.lname AS employee_name,
				   a.work_date, a.status, a.check_in, a.check_out, a.overtime_hours,
				   a.created_at, a.updated_at
			FROM attendance a
			JOIN employees e ON e.id = a.employee_id
			WHERE a.employee_id = $1
			  AND DATE_TRUNC('month', a.work_date) = DATE_TRUNC('month', $2::date)
			ORDER BY a.work_date;
		`
		rows, err = a.db.Query(ctx, query, empID, monthTime.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}

		// Start/End range query
	} else if start != "" && end != "" {
		startDate, err := time.Parse("2006-01-02", start)
		if err != nil {
			return nil, fmt.Errorf("invalid start date format, expected YYYY-MM-DD")
		}
		endDate, err := time.Parse("2006-01-02", end)
		if err != nil {
			return nil, fmt.Errorf("invalid end date format, expected YYYY-MM-DD")
		}

		query = `
			SELECT a.id, a.employee_id, e.fname || ' ' || e.lname AS employee_name,
				   a.work_date, a.status, a.check_in, a.check_out, a.overtime_hours,
				   a.created_at, a.updated_at
			FROM attendance a
			JOIN employees e ON e.id = a.employee_id
			WHERE a.employee_id = $1
			  AND a.work_date BETWEEN $2::date AND $3::date
			ORDER BY a.work_date;
		`
		rows, err = a.db.Query(ctx, query, empID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}

	} else {
		query = `
			SELECT a.id, a.employee_id, e.fname || ' ' || e.lname AS employee_name,
				   a.work_date, a.status, a.check_in, a.check_out, a.overtime_hours,
				   a.created_at, a.updated_at
			FROM attendance a
			JOIN employees e ON e.id = a.employee_id
			WHERE a.employee_id = $1
			ORDER BY a.work_date;
		`
		rows, err = a.db.Query(ctx, query, empID)
		if err != nil {
			return nil, err
		}
	}

	defer rows.Close()

	// Initialize calendar and attendance slice
	calendar := &models.EmployeeCalendar{
		Attendance: []*models.Attendance{},
	}

	for rows.Next() {
		var a models.Attendance
		var employeeName string

		err = rows.Scan(
			&a.ID, &a.EmployeeID, &employeeName,
			&a.WorkDate, &a.Status, &a.CheckIn, &a.CheckOut, &a.OvertimeHours,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		a.WorkDateStr = a.WorkDate.Format("2006-01-02")
		if !a.CheckIn.IsZero() {
			a.CheckInStr = a.CheckIn.Format("15:04")
		}
		if !a.CheckOut.IsZero() {
			a.CheckOutStr = a.CheckOut.Format("15:04")
		}
		calendar.EmployeeID = a.EmployeeID
		calendar.EmployeeName = employeeName
		calendar.Attendance = append(calendar.Attendance, &a)
	}

	// Check for iteration errors
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Set calendar month for reference
	if month != "" {
		calendar.Month = month
	} else if start != "" {
		calendar.Month = start[:7] // YYYY-MM from start date
	}

	return calendar, nil
}

// ----------------- SUMMARY -----------------

func (a *AttendanceRepo) GetEmployeeSummary(ctx context.Context, employeeID string, month string) (*models.AttendanceSummary, error) {
	query := `
		SELECT a.employee_id, e.fname || ' ' || e.lname AS employee_name,
		       COUNT(*) FILTER (WHERE a.status = 'Present') AS present_days,
		       COUNT(*) FILTER (WHERE a.status = 'Absent') AS absent_days,
		       COUNT(*) FILTER (WHERE a.status = 'Leave') AS leave_days,
		       COUNT(*) AS total_working_days,
		       COALESCE(SUM(a.overtime_hours), 0) AS total_overtime_hours
		FROM attendance a
		JOIN employees e ON e.id = a.employee_id
		WHERE a.employee_id = $1
		  AND DATE_TRUNC('month', a.work_date) = DATE_TRUNC('month', TO_DATE($2, 'YYYY-MM'))
		GROUP BY a.employee_id, e.fname, e.lname;
	`

	var s models.AttendanceSummary
	err := a.db.QueryRow(ctx, query, employeeID, month).Scan(
		&s.EmployeeID, &s.EmployeeName, &s.PresentDays,
		&s.AbsentDays, &s.LeaveDays, &s.TotalWorkingDays, &s.TotalOvertimeHours,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (a *AttendanceRepo) GetBatchSummary(ctx context.Context, month, start, end string) ([]models.AttendanceSummary, error) {
	var query string
	var rows pgx.Rows
	var err error

	if month != "" {
		query = `
			SELECT a.employee_id, e.fname || ' ' || e.lname AS employee_name,
			       COUNT(*) FILTER (WHERE a.status = 'Present') AS present_days,
			       COUNT(*) FILTER (WHERE a.status = 'Absent') AS absent_days,
			       COUNT(*) FILTER (WHERE a.status = 'Leave') AS leave_days,
			       COUNT(*) AS total_working_days,
			       COALESCE(SUM(a.overtime_hours), 0) AS total_overtime_hours
			FROM attendance a
			JOIN employees e ON e.id = a.employee_id
			WHERE DATE_TRUNC('month', a.work_date) = DATE_TRUNC('month', TO_DATE($1, 'YYYY-MM'))
			GROUP BY a.employee_id, e.fname, e.lname
			ORDER BY employee_name;
		`
		rows, err = a.db.Query(ctx, query, month)
	} else if start != "" && end != "" {
		query = `
			SELECT a.employee_id, e.fname || ' ' || e.lname AS employee_name,
			       COUNT(*) FILTER (WHERE a.status = 'Present') AS present_days,
			       COUNT(*) FILTER (WHERE a.status = 'Absent') AS absent_days,
			       COUNT(*) FILTER (WHERE a.status = 'Leave') AS leave_days,
			       COUNT(*) AS total_working_days,
			       COALESCE(SUM(a.overtime_hours), 0) AS total_overtime_hours
			FROM attendance a
			JOIN employees e ON e.id = a.employee_id
			WHERE a.work_date BETWEEN $1 AND $2
			GROUP BY a.employee_id, e.fname, e.lname
			ORDER BY employee_name;
		`
		rows, err = a.db.Query(ctx, query, start, end)
	} else {
		return nil, errors.New("either month or start/end date required")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.AttendanceSummary
	for rows.Next() {
		var s models.AttendanceSummary
		err := rows.Scan(&s.EmployeeID, &s.EmployeeName, &s.PresentDays, &s.AbsentDays, &s.LeaveDays, &s.TotalWorkingDays, &s.TotalOvertimeHours)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}
