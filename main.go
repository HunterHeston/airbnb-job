package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// JobPosting holds basic info for a job.
type JobPosting struct {
	Title string
	URL   string
}

func main() {
	// Base URL – note that the page number is appended at the end.
	// (You can adjust the URL if you prefer the /page/2/ format.)
	baseURL := "https://careers.airbnb.com/positions/?_departments=engineering&_offices=united-states&_paged="
	var allJobs []JobPosting

	page := 1
	for {
		url := fmt.Sprintf("%s%d", baseURL, page)
		fmt.Printf("Fetching page %d: %s\n", page, url)

		resp, err := http.Get(url)
		if err != nil {
			log.Fatalf("Error fetching page %d: %v", page, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Non-200 HTTP status on page %d: %d", page, resp.StatusCode)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatalf("Error parsing HTML on page %d: %v", page, err)
		}

		// Select all job items. Each job posting is contained in a <li> inside
		// <ul class="job-list" role="list">.
		jobItems := doc.Find("ul.job-list li[role='listitem']")
		if jobItems.Length() == 0 {
			fmt.Println("No job listings found on this page; ending pagination.")
			break
		}

		jobItems.Each(func(i int, s *goquery.Selection) {
			// The job title and URL are found in the <h3 class="text-size-4"> element's <a> tag.
			jobLink := s.Find("h3.text-size-4 a")
			title := strings.TrimSpace(jobLink.Text())
			link, exists := jobLink.Attr("href")
			if !exists {
				link = ""
			}

			// Filter for midlevel Software Engineer positions:
			// Must contain "Software Engineer" but not "Senior" or "Staff".
			if strings.Contains(title, "Software Engineer") &&
				!strings.Contains(title, "Senior") &&
				!strings.Contains(title, "Staff") &&
				!strings.Contains(title, "Sr.") &&
				!strings.Contains(title, "Principal") &&
				!strings.Contains(title, "Android") &&
				!strings.Contains(title, "iOS") {
				allJobs = append(allJobs, JobPosting{
					Title: title,
					URL:   link,
				})
			}
		})

		// If fewer than 10 job items are found on the page, assume it's the last page.
		if jobItems.Length() < 10 {
			fmt.Println("Fewer than 10 job items found; likely the last page.")
			break
		}

		page++
	}

	// Print the found midlevel Software Engineer positions.
	fmt.Printf("\nFound %d midlevel Software Engineer positions:\n", len(allJobs))
	for _, job := range allJobs {
		fmt.Printf("- %s (%s)\n", job.Title, job.URL)
	}

	sendDailyJobEmail(allJobs)
}

// sendDailyJobEmail composes and sends an email with the list of job postings.
// It uses Gmail's SMTP server. Make sure to use an app password or OAuth2 for Gmail.
func sendDailyJobEmail(jobPostings []JobPosting) error {
	from := os.Getenv("FROM_EMAIL")
	to := os.Getenv("TO_EMAIL")
	password := os.Getenv("GOOGLE_APP_PASSWORD")
	smtpHost := "smtp.gmail.com"
	smtpPort := "587" // TLS port

	// Build the email subject and body.
	subject := "Daily Job Postings"
	var body strings.Builder

	if len(jobPostings) == 0 {
		body.WriteString("Hello,\n\nNo current job postings found today.\n")
	} else {
		body.WriteString("Hello,\n\nHere are today's midlevel Software Engineer job postings:\n\n")
	}

	for _, job := range jobPostings {
		body.WriteString(fmt.Sprintf("- %s: %s\n", job.Title, job.URL))
	}

	body.WriteString("\n You can find more job postings at https://careers.airbnb.com/positions/?_departments=engineering&_offices=united-states\n")

	body.WriteString("\nBest regards,\nYour Job Scraper")

	// Construct the full email message including headers.
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		from, to, subject, body.String())

	// Set up authentication information.
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Send the email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(message))
	if err != nil {
		return err
	}
	return nil
}
