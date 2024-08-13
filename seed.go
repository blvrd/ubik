package main

import (
	"time"
	"github.com/google/uuid"
)

var seedIssues = []Issue{
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "emma@techco.com",
		Title:     "Memory leak in background worker process",
		CreatedAt: time.Now().UTC(),
		Description: `
Our background worker process is experiencing a memory leak. The process starts with normal memory usage but gradually increases over time, eventually causing the worker to crash.

Steps to reproduce:
1. Start the background worker
2. Monitor memory usage over 24 hours
3. Observe steady increase in memory consumption

We need to identify the source of the leak and implement a fix.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "devops@techco.com",
				content: "I've added memory profiling to the worker. Will analyze the results and report back.",
			},
			{
				author:  "emma@techco.com",
				content: "Thanks for looking into this. Let me know if you need any additional information.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "alex@devfirm.com",
		Title:     "API rate limiting not working correctly",
		CreatedAt: time.Now().UTC(),
		Description: `
The rate limiting on our API is not functioning as expected. Users are able to make more requests than the specified limit within the time window.

Steps to reproduce:
1. Set up a test client to make rapid API requests
2. Observe that more than the allowed number of requests are successful
3. Check server logs for rate limiting entries

Expected: Requests beyond the limit should be rejected
Actual: All requests are being accepted
        `,
		Status: 1,
		Comments: []Comment{
			{
				author:  "backend@devfirm.com",
				content: "I've identified the issue. Our Redis cache for storing rate limit data wasn't being updated correctly. Working on a fix now.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "ux@designstudio.com",
		Title:     "Mobile responsive design breaks on iPhone 12",
		CreatedAt: time.Now().UTC(),
		Description: `
Our responsive design is not rendering correctly on iPhone 12 devices. The layout is broken and some elements are overlapping.

Steps to reproduce:
1. Open our website on an iPhone 12
2. Navigate to the product listing page
3. Observe misaligned elements and text overflow

This issue seems to be specific to iPhone 12 models and doesn't occur on other iOS devices we've tested.
        `,
		Status: 3,
		Comments: []Comment{
			{
				author:  "frontend@designstudio.com",
				content: "I've identified the cause. It's related to the new screen resolution on iPhone 12. Working on a CSS fix.",
			},
			{
				author:  "ux@designstudio.com",
				content: "Great, thanks for the quick response. Please let me know when a fix is ready for testing.",
			},
			{
				author:  "frontend@designstudio.com",
				content: "Fix has been implemented and pushed to staging. Please review and let me know if any further adjustments are needed.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "security@bigcorp.com",
		Title:     "Potential SQL injection vulnerability in search function",
		CreatedAt: time.Now().UTC(),
		Description: `
During a routine security audit, we identified a potential SQL injection vulnerability in the product search function.

Steps to reproduce:
1. Navigate to the product search page
2. Enter the following in the search box: '; DROP TABLE users; --
3. Observe that the application throws an error instead of handling the input safely

This needs to be addressed urgently to prevent potential data breaches.
        `,
		Status: 1,
		Comments: []Comment{
			{
				author:  "backend@bigcorp.com",
				content: "Thanks for flagging this. I'm implementing prepared statements for all database queries to prevent SQL injection. Will push a fix for review shortly.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "pm@saasplatform.com",
		Title:     "User authentication fails intermittently",
		CreatedAt: time.Now().UTC(),
		Description: `
We're receiving reports from users that they're occasionally unable to log in, even with correct credentials. The issue seems to resolve itself after a few minutes.

Steps to reproduce:
1. Attempt to log in with valid credentials
2. If login fails, wait a few minutes and try again
3. Login should succeed on second or third attempt

This is happening to approximately 5% of login attempts.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "backend@saasplatform.com",
				content: "I've added additional logging to the authentication process. Will monitor and analyze logs to identify any patterns or issues.",
			},
			{
				author:  "devops@saasplatform.com",
				content: "Could this be related to cache inconsistencies? I'll check our Redis cluster for any replication delays.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "data@analyticsfirm.com",
		Title:     "Incorrect data aggregation in monthly reports",
		CreatedAt: time.Now().UTC(),
		Description: `
The monthly aggregated reports are showing inconsistent data compared to daily reports. The discrepancy is approximately 3-5% and appears to be systematic.

Steps to reproduce:
1. Generate a daily report for each day in the last month
2. Sum the totals from the daily reports
3. Generate a monthly report for the same period
4. Compare the totals - they should match but don't

This is affecting our clients' decision-making processes and needs to be resolved quickly.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "backend@analyticsfirm.com",
				content: "I'm reviewing the aggregation queries. Initial investigation suggests we might be double-counting some events in the monthly rollup. Will update once I have more information.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "support@cloudservice.com",
		Title:     "File upload failing for files over 50MB",
		CreatedAt: time.Now().UTC(),
		Description: `
Users are reporting that file uploads are failing for any file larger than 50MB. Our system should support uploads up to 200MB.

Steps to reproduce:
1. Log in to the user dashboard
2. Attempt to upload a file larger than 50MB
3. Observe that the upload fails with a generic error message

This is blocking some of our enterprise customers from using a key feature of our platform.
        `,
		Status: 1,
		Comments: []Comment{
			{
				author:  "devops@cloudservice.com",
				content: "I've checked our Nginx configuration and found that the client_max_body_size was set to 50M. Updating to 200M and will deploy the change.",
			},
			{
				author:  "backend@cloudservice.com",
				content: "We should also update our client-side validation to match the new limit. I'll make those changes.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "ux@mobileapp.com",
		Title:     "App crashes when accessing camera on Android 11",
		CreatedAt: time.Now().UTC(),
		Description: `
Our mobile app is crashing when users try to access the camera feature on devices running Android 11. This doesn't occur on other Android versions.

Steps to reproduce:
1. Install the app on an Android 11 device
2. Navigate to the 'Take Photo' feature
3. App crashes immediately when the camera is initialized

This is a critical feature for our app and needs to be fixed ASAP.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "mobile@mobileapp.com",
				content: "I've reproduced the issue. It seems to be related to the new scoped storage changes in Android 11. I'm working on updating our camera access implementation to comply with the new requirements.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "pm@ecommerce.com",
		Title:     "Checkout process hangs at payment step",
		CreatedAt: time.Now().UTC(),
		Description: `
Some users are reporting that the checkout process hangs indefinitely after entering payment information. This is resulting in lost sales.

Steps to reproduce:
1. Add items to cart
2. Proceed to checkout
3. Enter shipping and billing information
4. On the payment page, enter card details and submit
5. Page appears to load indefinitely without completing the transaction

This is happening sporadically, affecting roughly 10% of transactions.
        `,
		Status: 1,
		Comments: []Comment{
			{
				author:  "backend@ecommerce.com",
				content: "I've added additional logging to the payment processing step. Will analyze logs from affected transactions to identify any patterns.",
			},
			{
				author:  "frontend@ecommerce.com",
				content: "Could this be a frontend issue? I'll check for any JavaScript errors or race conditions in the payment form submission.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "devops@saas.com",
		Title:     "Elasticsearch cluster running out of disk space",
		CreatedAt: time.Now().UTC(),
		Description: `
Our Elasticsearch cluster is rapidly running out of disk space, much faster than expected based on our data growth projections.

Steps to reproduce:
1. Monitor disk usage on Elasticsearch nodes
2. Observe abnormally rapid increase in used space
3. Project time until disks are full (currently estimating 72 hours until impact)

We need to identify the cause of the increased disk usage and implement a solution quickly to avoid service disruption.
        `,
		Status: 1,
		Comments: []Comment{
			{
				author:  "backend@saas.com",
				content: "I'm investigating our index management. We might need to optimize our index lifecycle policies or increase our pruning of old data.",
			},
			{
				author:  "devops@saas.com",
				content: "As a temporary measure, I'm provisioning additional nodes to the cluster to buy us some time. Will coordinate with backend team on a permanent solution.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "security@fintech.com",
		Title:     "Possible data leak in API response",
		CreatedAt: time.Now().UTC(),
		Description: `
We've identified a potential data leak in one of our API endpoints. The endpoint is returning more user data than it should, possibly exposing sensitive information.

Steps to reproduce:
1. Authenticate as a regular user
2. Make a GET request to /api/v1/user/profile
3. Observe that the response includes fields like 'ssn' and 'tax_id' which should not be exposed

This needs to be addressed immediately to ensure we're not violating any data protection regulations.
        `,
		Status: 1,
		Comments: []Comment{
			{
				author:  "backend@fintech.com",
				content: "I've identified the issue. We're not properly filtering the user object before sending it in the API response. Implementing a fix now and will also audit other endpoints for similar issues.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "qa@gamedev.com",
		Title:     "Game freezes during level transition on low-end devices",
		CreatedAt: time.Now().UTC(),
		Description: `
Our mobile game is freezing during level transitions on low-end Android devices. This is causing a poor user experience and increasing our churn rate.

Steps to reproduce:
1. Install the game on a low-end Android device (e.g., 2GB RAM, older processor)
2. Play through the first level
3. When transitioning to the second level, game freezes for 10-15 seconds

The issue is not present on high-end devices or iOS.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "dev@gamedev.com",
				content: "I've reproduced the issue. It seems to be related to asset loading during level transition. I'm working on implementing asynchronous loading to reduce the impact on the main thread.",
			},
			{
				author:  "qa@gamedev.com",
				content: "Thanks for the update. Please let me know when you have a build ready for testing. I'll verify on a range of low-end devices.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "ux@webapp.com",
		Title:     "WCAG compliance issues on main dashboard",
		CreatedAt: time.Now().UTC(),
		Description: `
An accessibility audit has revealed several WCAG 2.1 compliance issues on our main dashboard, potentially making the app unusable for users with disabilities.

Key issues:
1. Insufficient color contrast on several text elements
2. Missing alt text on icons and images
3. Keyboard navigation not working for some interactive elements

We need to address these issues to improve accessibility and avoid potential legal issues.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "frontend@webapp.com",
				content: "I'm working through the issues one by one. Have resolved the color contrast problems and am now adding proper alt text to all images and icons.",
			},
			{
				author:  "ux@webapp.com",
				content: "Great progress. For the keyboard navigation, make sure we're using proper ARIA attributes where necessary.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "devops@streaming.com",
		Title:     "CDN caching issue causing stale content delivery",
		CreatedAt: time.Now().UTC(),
		Description: `
Users are reporting that they're sometimes seeing outdated content, even after we've pushed updates to our site. This appears to be a caching issue with our CDN.

Steps to reproduce:
1. Push an update to the production site
2. Immediately visit the site from different geographic locations
3. Observe that some locations are still seeing the old content

This is causing confusion among our users and needs to be resolved.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "backend@streaming.com",
				content: "I've reviewed our cache invalidation process. We're not properly purging the CDN cache after content updates. I'm implementing a webhook to automatically purge relevant cache entries on content changes.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "pm@adtech.com",
		Title:     "Discrepancy in ad impression counting",
		CreatedAt: time.Now().UTC(),
		Description: `
We've noticed a discrepancy between our ad impression counts and those reported by our clients' systems. Our numbers are consistently 5-8% higher.

Steps to reproduce:
1. Run an ad campaign for 24 hours
2. Compare impression counts from our dashboard with client-provided data
3. Calculate the percentage difference

This discrepancy is causing billing disputes and needs to be resolved to maintain client trust.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "backend@adtech.com",
				content: "I'm investigating our impression counting logic. Initial findings suggest we might be double-counting some impressions due to a race condition in our event processing pipeline.",
			},
			{
				author:  "data@adtech.com",
				content: "I'll work on reconciling our data with client data to identify patterns in the discrepancies. This might help pinpoint the source of the issue.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "security@bank.com",
		Title:     "Potential timing attack vulnerability in login process",
		CreatedAt: time.Now().UTC(),
		Description: `
Our security team has identified a potential timing attack vulnerability in our login process. The response time for login attempts varies noticeably depending on whether the username exists or not.

Steps to reproduce:
1. Attempt to log in with a non-existent username
2. Attempt to log in with an existing username (with wrong password)
3. Compare response times

This could potentially be exploited to enumerate valid usernames, which is a security risk.
        `,
		Status: 1,
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "devops@cloudprovider.com",
		Title:     "Intermittent network timeouts in Kubernetes cluster",
		CreatedAt: time.Now().UTC(),
		Description: `
We're experiencing intermittent network timeouts between pods in our Kubernetes cluster. This is causing sporadic failures in inter-service communication.

Steps to reproduce:
1. Deploy our microservices stack to the affected cluster
2. Run our integration test suite
3. Observe that approximately 5% of tests fail due to network timeouts

The issue seems to occur more frequently during high-load periods.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "netops@cloudprovider.com",
				content: "I've started investigating the network configuration. Initial findings suggest it might be related to kube-proxy settings. Will update once I have more information.",
			},
			{
				author:  "devops@cloudprovider.com",
				content: "Thanks for looking into this. I've increased the logging on affected services to gather more data on the timeouts.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "frontend@webapp.com",
		Title:     "React component re-rendering excessively",
		CreatedAt: time.Now().UTC(),
		Description: `
We've identified a performance issue in our React application where a specific component is re-rendering much more frequently than necessary, causing noticeable UI lag.

Steps to reproduce:
1. Navigate to the user dashboard
2. Open the activity feed component
3. Scroll through the feed
4. Observe significant frame drops and lag

This is particularly noticeable on lower-end devices.
        `,
		Status: 1,
		Comments: []Comment{
			{
				author:  "frontend@webapp.com",
				content: "I've started profiling the component. It looks like we're not memoizing some expensive computations, causing unnecessary re-renders. Working on a fix using useMemo and useCallback.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "data@analytics.com",
		Title:     "Inconsistent results in A/B testing framework",
		CreatedAt: time.Now().UTC(),
		Description: `
Our A/B testing framework is producing inconsistent results. We're seeing statistically significant differences in metrics between A and B groups even when no changes have been made.

Steps to reproduce:
1. Set up an A/B test with identical variations
2. Run the test for at least 7 days
3. Analyze the results
4. Observe unexpected differences between groups

This is undermining our ability to make data-driven decisions.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "backend@analytics.com",
				content: "I'm reviewing our user segmentation logic. There might be a bias in how we're assigning users to groups. Will update once I've investigated further.",
			},
			{
				author:  "data@analytics.com",
				content: "Thanks for looking into this. I'll also double-check our statistical analysis methods to ensure we're not making any incorrect assumptions.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "security@fintech.com",
		Title:     "Potential XSS vulnerability in user profile page",
		CreatedAt: time.Now().UTC(),
		Description: `
A security audit has revealed a potential Cross-Site Scripting (XSS) vulnerability on the user profile page. User-supplied content is being rendered without proper sanitization.

Steps to reproduce:
1. Log in to a user account
2. Edit the profile description
3. Insert a malicious script tag, e.g., <script>alert('XSS')</script>
4. Save the profile and view it
5. Observe that the script executes

This vulnerability could allow attackers to inject malicious scripts into our site.
        `,
		Status: 1,
		Comments: []Comment{
			{
				author:  "frontend@fintech.com",
				content: "I'm implementing proper input sanitization and output encoding to prevent XSS attacks. Will also conduct a broader security review of our frontend code.",
			},
			{
				author:  "security@fintech.com",
				content: "Great, thanks for the quick response. Please let me know when the fix is ready for testing. We'll need to do a thorough security review before deploying.",
			},
		},
	},
	{
		Id:        uuid.NewString(),
    Shortcode: StringToShortcode(uuid.NewString()),
		Author:    "pm@saas.com",
		Title:     "Billing cycle not aligning with subscription dates",
		CreatedAt: time.Now().UTC(),
		Description: `
We've received reports from customers that their billing cycles are not aligning with their subscription dates. This is causing confusion and in some cases, incorrect billing.

Steps to reproduce:
1. Create a new subscription starting on the 15th of the month
2. Observe that the first bill is generated on the 1st of the next month
3. Subsequent bills are also generated on the 1st, not the 15th

This misalignment is affecting our revenue recognition and causing customer complaints.
        `,
		Status: 2,
		Comments: []Comment{
			{
				author:  "backend@saas.com",
				content: "I'm investigating our billing system. It appears we're not correctly handling mid-month subscription starts. I'll implement a fix to ensure billing dates align with subscription start dates.",
			},
			{
				author:  "finance@saas.com",
				content: "Once the fix is implemented, we'll need to audit affected accounts and issue corrections where necessary. I'll prepare a plan for this.",
			},
		},
	},
}
