package email

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// InvitationTemplateData contains values for rendering interview invitations.
type InvitationTemplateData struct {
	CandidateName    string
	JobTitle         string
	JobCode          string
	StageLabel       string
	InterviewType    string // ONLINE, ONSITE, PHONE
	ScheduledStart   time.Time
	ScheduledEnd     time.Time
	MeetingURL       string
	Location         string
	InterviewerNames []string
	InterviewID      string
	CandidateID      string
	BaseURL          string
}

// ReminderTemplateData contains values for rendering interview reminders.
type ReminderTemplateData struct {
	RecipientName  string
	CandidateName  string
	JobTitle       string
	JobCode        string
	StageLabel     string
	InterviewType  string
	ScheduledStart time.Time
	ScheduledEnd   time.Time
	MeetingURL     string
	Location       string
	InterviewID    string
	BaseURL        string
}

func formatJakartaTime(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return t.In(loc).Format("15:04")
}

func formatJakartaDate(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return t.In(loc).Format("Monday, 02 January 2006")
}

// BuildCandidateInvitation renders the HTML and plain text for a candidate invitation.
func BuildCandidateInvitation(data InvitationTemplateData) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("Interview Invitation — %s at HireScope", data.StageLabel)

	dateStr := formatJakartaDate(data.ScheduledStart)
	timeStr := fmt.Sprintf("%s – %s WIB", formatJakartaTime(data.ScheduledStart), formatJakartaTime(data.ScheduledEnd))

	safeCandidate := html.EscapeString(data.CandidateName)
	safeJob := html.EscapeString(data.JobTitle)
	safeJobCode := html.EscapeString(data.JobCode)
	safeStage := html.EscapeString(data.StageLabel)
	safeType := html.EscapeString(data.InterviewType)
	safeLocation := html.EscapeString(data.Location)

	var interviewerList []string
	for _, n := range data.InterviewerNames {
		interviewerList = append(interviewerList, html.EscapeString(n))
	}
	interviewersDisplay := strings.Join(interviewerList, ", ")
	if interviewersDisplay == "" {
		interviewersDisplay = "HireScope Interview Team"
	}

	rawInterviewersDisplay := strings.Join(data.InterviewerNames, ", ")
	if rawInterviewersDisplay == "" {
		rawInterviewersDisplay = "HireScope Interview Team"
	}

	calendarURL := fmt.Sprintf("%s/api/v1/interviews/%s/ics", strings.TrimRight(data.BaseURL, "/"), data.InterviewID)

	var modalityHTML string
	var modalityText string
	if data.InterviewType == "ONLINE" && data.MeetingURL != "" {
		sanitizedURL := strings.ReplaceAll(strings.ReplaceAll(data.MeetingURL, "\r", ""), "\n", "")
		cleanURL := html.EscapeString(sanitizedURL)
		modalityHTML = fmt.Sprintf(`
			<tr>
				<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Meeting Link</td>
				<td style="padding: 8px 0; font-size: 13px;">
					<a href="%s" style="display: inline-block; padding: 6px 14px; background-color: #2563eb; color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 12px; margin-bottom: 4px;">Join Meeting</a><br>
					<span style="color: #64748b; font-size: 11px; word-break: break-all;">%s</span>
				</td>
			</tr>`, cleanURL, cleanURL)
		modalityText = fmt.Sprintf("Join Meeting: %s\n", sanitizedURL)
	} else if data.InterviewType == "ONSITE" && data.Location != "" {
		modalityHTML = fmt.Sprintf(`
			<tr>
				<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Location</td>
				<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 500;">%s</td>
			</tr>`, safeLocation)
		modalityText = fmt.Sprintf("Location: %s\n", data.Location)
	}

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f8fafc; margin: 0; padding: 24px; color: #0f172a;">
	<table width="100%%" border="0" cellspacing="0" cellpadding="0" style="max-width: 580px; margin: 0 auto; background-color: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.05);">
		<tr>
			<td style="padding: 24px; border-bottom: 1px solid #e2e8f0; background-color: #f8fafc;">
				<h2 style="margin: 0; font-size: 18px; font-weight: 700; color: #0f172a; font-family: monospace; letter-spacing: -0.5px;">HIRESCOPE</h2>
				<p style="margin: 4px 0 0 0; font-size: 12px; color: #64748b;">Recruitment &amp; Assessment Management</p>
			</td>
		</tr>
		<tr>
			<td style="padding: 24px;">
				<p style="font-size: 14px; margin: 0 0 16px 0; color: #334155;">Hello <strong>%s</strong>,</p>
				<p style="font-size: 14px; margin: 0 0 20px 0; color: #334155; line-height: 1.5;">You are invited to an interview session for the position of <strong>%s</strong> (%s).</p>
				
				<table width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; padding: 14px; margin-bottom: 20px;">
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Stage</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 600;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Date</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 600;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Time</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 600;">%s (GMT+7)</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Format</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 500;">%s</td>
					</tr>
					%s
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Interviewers</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px;">%s</td>
					</tr>
				</table>

				<div style="margin: 20px 0; padding: 12px 16px; background-color: #eff6ff; border: 1px solid #bfdbfe; border-radius: 6px; font-size: 12px; color: #1e40af;">
					📅 <strong>Calendar:</strong> You can add this session to your calendar: <a href="%s" style="color: #2563eb; font-weight: 600; text-decoration: underline;">Download .ics Event File</a>
				</div>

				<p style="font-size: 13px; color: #64748b; line-height: 1.5; margin: 16px 0 0 0;">Please contact our recruitment team if you need to reschedule or have any questions beforehand.</p>
			</td>
		</tr>
		<tr>
			<td style="padding: 16px 24px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; font-size: 11px; color: #94a3b8; text-align: center;">
				HireScope Recruitment Operations &bull; Automated notification
			</td>
		</tr>
	</table>
</body>
</html>`, html.EscapeString(subject), safeCandidate, safeJob, safeJobCode, safeStage, dateStr, timeStr, safeType, modalityHTML, interviewersDisplay, calendarURL)

	textBody = fmt.Sprintf(`Hello %s,

You are invited to an interview session for:
%s (%s)

Interview Details:
Stage: %s
Date: %s
Time: %s
Timezone: Asia/Jakarta (GMT+7)
Format: %s
%sInterviewers: %s

Add to Calendar (.ics): %s

Please contact our recruitment team if you need to discuss the schedule.

Regards,
HireScope Recruitment Team
`, data.CandidateName, data.JobTitle, data.JobCode, data.StageLabel, dateStr, timeStr, data.InterviewType, modalityText, rawInterviewersDisplay, calendarURL)

	return subject, htmlBody, textBody
}

// BuildInterviewerInvitation renders HTML and plain text for an interviewer notification.
func BuildInterviewerInvitation(interviewerName string, data InvitationTemplateData) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("Interview Scheduled — %s — %s", data.CandidateName, data.StageLabel)

	dateStr := formatJakartaDate(data.ScheduledStart)
	timeStr := fmt.Sprintf("%s – %s WIB", formatJakartaTime(data.ScheduledStart), formatJakartaTime(data.ScheduledEnd))

	safeInterviewer := html.EscapeString(interviewerName)
	safeCandidate := html.EscapeString(data.CandidateName)
	safeJob := html.EscapeString(data.JobTitle)
	safeJobCode := html.EscapeString(data.JobCode)
	safeStage := html.EscapeString(data.StageLabel)
	safeType := html.EscapeString(data.InterviewType)
	safeLocation := html.EscapeString(data.Location)

	var interviewerList []string
	for _, n := range data.InterviewerNames {
		interviewerList = append(interviewerList, html.EscapeString(n))
	}
	interviewersDisplay := strings.Join(interviewerList, ", ")
	rawInterviewersDisplay := strings.Join(data.InterviewerNames, ", ")

	detailURL := fmt.Sprintf("%s/interviews/%s", strings.TrimRight(data.BaseURL, "/"), data.InterviewID)
	calendarURL := fmt.Sprintf("%s/api/v1/interviews/%s/ics", strings.TrimRight(data.BaseURL, "/"), data.InterviewID)

	var modalityHTML string
	var modalityText string
	if data.InterviewType == "ONLINE" && data.MeetingURL != "" {
		sanitizedURL := strings.ReplaceAll(strings.ReplaceAll(data.MeetingURL, "\r", ""), "\n", "")
		cleanURL := html.EscapeString(sanitizedURL)
		modalityHTML = fmt.Sprintf(`
			<tr>
				<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Meeting Link</td>
				<td style="padding: 8px 0; font-size: 13px;">
					<a href="%s" style="color: #2563eb; font-weight: 600; text-decoration: underline;">%s</a>
				</td>
			</tr>`, cleanURL, cleanURL)
		modalityText = fmt.Sprintf("Online Meeting: %s\n", sanitizedURL)
	} else if data.InterviewType == "ONSITE" && data.Location != "" {
		modalityHTML = fmt.Sprintf(`
			<tr>
				<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Location</td>
				<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 500;">%s</td>
			</tr>`, safeLocation)
		modalityText = fmt.Sprintf("Location: %s\n", data.Location)
	}

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f8fafc; margin: 0; padding: 24px; color: #0f172a;">
	<table width="100%%" border="0" cellspacing="0" cellpadding="0" style="max-width: 580px; margin: 0 auto; background-color: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.05);">
		<tr>
			<td style="padding: 24px; border-bottom: 1px solid #e2e8f0; background-color: #f8fafc;">
				<h2 style="margin: 0; font-size: 18px; font-weight: 700; color: #0f172a; font-family: monospace; letter-spacing: -0.5px;">HIRESCOPE</h2>
				<p style="margin: 4px 0 0 0; font-size: 12px; color: #64748b;">Interviewer Assignment Notification</p>
			</td>
		</tr>
		<tr>
			<td style="padding: 24px;">
				<p style="font-size: 14px; margin: 0 0 16px 0; color: #334155;">Hello <strong>%s</strong>,</p>
				<p style="font-size: 14px; margin: 0 0 20px 0; color: #334155; line-height: 1.5;">You have been assigned as an interviewer for candidate <strong>%s</strong>.</p>
				
				<table width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; padding: 14px; margin-bottom: 20px;">
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Candidate</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 600;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Position</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 500;">%s (%s)</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Stage</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 600;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Date &amp; Time</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 600;">%s, %s (GMT+7)</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Format</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 500;">%s</td>
					</tr>
					%s
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Interviewers</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px;">%s</td>
					</tr>
				</table>

				<div style="margin: 20px 0; display: flex; gap: 10px;">
					<a href="%s" style="display: inline-block; padding: 8px 16px; background-color: #0f172a; color: #ffffff; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 13px;">View Interview in HireScope &rarr;</a>
				</div>

				<p style="font-size: 12px; color: #64748b; margin: 12px 0 0 0;">📅 Add to Calendar: <a href="%s" style="color: #2563eb; text-decoration: underline;">Download .ics Event</a></p>
			</td>
		</tr>
		<tr>
			<td style="padding: 16px 24px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; font-size: 11px; color: #94a3b8; text-align: center;">
				HireScope Recruitment Operations &bull; Automated notification
			</td>
		</tr>
	</table>
</body>
</html>`, html.EscapeString(subject), safeInterviewer, safeCandidate, safeCandidate, safeJob, safeJobCode, safeStage, dateStr, timeStr, safeType, modalityHTML, interviewersDisplay, detailURL, calendarURL)

	textBody = fmt.Sprintf(`Hello %s,

You have been assigned to an upcoming interview session.

Candidate: %s
Position: %s (%s)
Stage: %s
Date: %s
Time: %s (Asia/Jakarta, GMT+7)
Format: %s
%sInterviewers: %s

View in HireScope: %s
Add to Calendar (.ics): %s

Please review the candidate profile in HireScope prior to the session.

Regards,
HireScope
`, interviewerName, data.CandidateName, data.JobTitle, data.JobCode, data.StageLabel, dateStr, timeStr, data.InterviewType, modalityText, rawInterviewersDisplay, detailURL, calendarURL)

	return subject, htmlBody, textBody
}

// BuildReminder renders HTML and plain text for an interview reminder.
func BuildReminder(data ReminderTemplateData) (subject, htmlBody, textBody string) {
	subject = fmt.Sprintf("Reminder — Interview Upcoming — %s", data.CandidateName)

	dateStr := formatJakartaDate(data.ScheduledStart)
	timeStr := fmt.Sprintf("%s – %s WIB", formatJakartaTime(data.ScheduledStart), formatJakartaTime(data.ScheduledEnd))

	safeRecipient := html.EscapeString(data.RecipientName)
	safeCandidate := html.EscapeString(data.CandidateName)
	safeJob := html.EscapeString(data.JobTitle)
	safeJobCode := html.EscapeString(data.JobCode)
	safeStage := html.EscapeString(data.StageLabel)
	safeType := html.EscapeString(data.InterviewType)
	safeLocation := html.EscapeString(data.Location)

	calendarURL := fmt.Sprintf("%s/api/v1/interviews/%s/ics", strings.TrimRight(data.BaseURL, "/"), data.InterviewID)

	var modalityHTML string
	var modalityText string
	if data.InterviewType == "ONLINE" && data.MeetingURL != "" {
		sanitizedURL := strings.ReplaceAll(strings.ReplaceAll(data.MeetingURL, "\r", ""), "\n", "")
		cleanURL := html.EscapeString(sanitizedURL)
		modalityHTML = fmt.Sprintf(`
			<tr>
				<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Meeting Link</td>
				<td style="padding: 8px 0; font-size: 13px;">
					<a href="%s" style="color: #2563eb; font-weight: 600; text-decoration: underline;">%s</a>
				</td>
			</tr>`, cleanURL, cleanURL)
		modalityText = fmt.Sprintf("Online Meeting: %s\n", sanitizedURL)
	} else if data.InterviewType == "ONSITE" && data.Location != "" {
		modalityHTML = fmt.Sprintf(`
			<tr>
				<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Location</td>
				<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 500;">%s</td>
			</tr>`, safeLocation)
		modalityText = fmt.Sprintf("Location: %s\n", data.Location)
	}

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f8fafc; margin: 0; padding: 24px; color: #0f172a;">
	<table width="100%%" border="0" cellspacing="0" cellpadding="0" style="max-width: 580px; margin: 0 auto; background-color: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.05);">
		<tr>
			<td style="padding: 24px; border-bottom: 1px solid #e2e8f0; background-color: #f8fafc;">
				<h2 style="margin: 0; font-size: 18px; font-weight: 700; color: #0f172a; font-family: monospace; letter-spacing: -0.5px;">HIRESCOPE</h2>
				<p style="margin: 4px 0 0 0; font-size: 12px; color: #64748b;">Upcoming Interview Reminder</p>
			</td>
		</tr>
		<tr>
			<td style="padding: 24px;">
				<p style="font-size: 14px; margin: 0 0 16px 0; color: #334155;">Hello <strong>%s</strong>,</p>
				<p style="font-size: 14px; margin: 0 0 20px 0; color: #334155; line-height: 1.5;">This is a reminder for your upcoming interview session.</p>
				
				<table width="100%%" border="0" cellspacing="0" cellpadding="0" style="background-color: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; padding: 14px; margin-bottom: 20px;">
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Candidate</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 600;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Position</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 500;">%s (%s)</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Stage</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 600;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Date &amp; Time</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 600;">%s, %s (GMT+7)</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #64748b; font-size: 13px; width: 140px;">Format</td>
						<td style="padding: 8px 0; color: #0f172a; font-size: 13px; font-weight: 500;">%s</td>
					</tr>
					%s
				</table>

				<p style="font-size: 12px; color: #64748b; margin: 12px 0 0 0;">📅 Add to Calendar: <a href="%s" style="color: #2563eb; text-decoration: underline;">Download .ics Event</a></p>
			</td>
		</tr>
		<tr>
			<td style="padding: 16px 24px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; font-size: 11px; color: #94a3b8; text-align: center;">
				HireScope Recruitment Operations &bull; Automated notification
			</td>
		</tr>
	</table>
</body>
</html>`, html.EscapeString(subject), safeRecipient, safeCandidate, safeJob, safeJobCode, safeStage, dateStr, timeStr, safeType, modalityHTML, calendarURL)

	textBody = fmt.Sprintf(`Hello %s,

This is a reminder for the upcoming interview session.

Candidate: %s
Position: %s (%s)
Stage: %s
Date: %s
Time: %s (Asia/Jakarta, GMT+7)
Format: %s
%s
Add to Calendar (.ics): %s

Regards,
HireScope
`, data.RecipientName, data.CandidateName, data.JobTitle, data.JobCode, data.StageLabel, dateStr, timeStr, data.InterviewType, modalityText, calendarURL)

	return subject, htmlBody, textBody
}
