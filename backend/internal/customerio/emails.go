package customerio

import "fmt"

func emailWrapper(content string) string {
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head><body style="margin:0;padding:0;background-color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f5;"><tr><td align="center" style="padding:40px 20px;"><table role="presentation" width="600" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:8px;max-width:600px;width:100%;"><tr><td style="padding:32px 40px;">` + content + `</td></tr><tr><td style="padding:16px 40px 32px;border-top:1px solid #e4e4e7;"><p style="margin:0;font-size:12px;color:#a1a1aa;text-align:center;">Star League Intelligence Command · <a href="https://starleagueintelligencecommand.com" style="color:#a1a1aa;">starleagueintelligencecommand.com</a></p></td></tr></table></td></tr></table></body></html>`
}

// WelcomeEmailHTML returns the welcome email body.
func WelcomeEmailHTML() string {
	return emailWrapper(`<h1 style="margin:0 0 16px;font-size:22px;color:#18181b;">Welcome to SLIC, MechWarrior</h1>
<p style="margin:0 0 20px;font-size:15px;color:#3f3f46;line-height:1.5;">You've just logged into the Star League Intelligence Command — a combat database for every BattleMech variant in existence.</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin-bottom:24px;">
<tr><td style="padding:8px 0;"><strong style="color:#18181b;">Combat Rating</strong><br><span style="font-size:14px;color:#52525b;">Monte Carlo sim scores for 4,227 variants</span></td></tr>
<tr><td style="padding:8px 0;border-top:1px solid #f4f4f5;"><strong style="color:#18181b;">List Builder</strong><br><span style="font-size:14px;color:#52525b;">Build tournament lists with BV tracking and shareable URLs</span></td></tr>
<tr><td style="padding:8px 0;border-top:1px solid #f4f4f5;"><strong style="color:#18181b;">Collection Tracker</strong><br><span style="font-size:14px;color:#52525b;">Track your physical miniatures across manufacturers</span></td></tr>
</table>
<table role="presentation" cellpadding="0" cellspacing="0" style="margin-bottom:24px;"><tr><td style="background-color:#5e6ad2;border-radius:6px;"><a href="https://starleagueintelligencecommand.com" style="display:inline-block;padding:12px 24px;color:#ffffff;text-decoration:none;font-weight:600;font-size:15px;">Build Your First List →</a></td></tr></table>
<p style="margin:0;font-size:15px;color:#3f3f46;">Good hunting.<br>— SLIC Command</p>`)
}

// FeedbackAckEmailHTML returns the feedback acknowledgment email body.
func FeedbackAckEmailHTML(issueURL string) string {
	linkHTML := ""
	if issueURL != "" {
		linkHTML = fmt.Sprintf(`<p style="margin:0 0 16px;font-size:15px;color:#3f3f46;line-height:1.5;">You can track your feedback here: <a href="%s" style="color:#5e6ad2;">%s</a></p>`, issueURL, issueURL)
	}
	return emailWrapper(`<h1 style="margin:0 0 16px;font-size:22px;color:#18181b;">We got your feedback</h1>
<p style="margin:0 0 16px;font-size:15px;color:#3f3f46;line-height:1.5;">Thanks for taking the time to send us feedback. We read every submission.</p>
` + linkHTML + `
<p style="margin:0 0 20px;font-size:15px;color:#3f3f46;line-height:1.5;">We're a small project built by BattleTech players for BattleTech players. Your input directly shapes what we build next.</p>
<p style="margin:0;font-size:15px;color:#3f3f46;">— SLIC Command</p>`)
}
