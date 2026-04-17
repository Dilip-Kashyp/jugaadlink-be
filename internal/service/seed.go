package service

import (
	"math/rand"
	"net/http"
	"time"
	"url-shortener/internal/config"
	"url-shortener/internal/models"
	"url-shortener/internal/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var sampleURLs = []struct {
	URL   string
	Title string
	Desc  string
	Image string
}{
	{"https://github.com", "GitHub", "Where the world builds software", "https://github.githubassets.com/images/modules/open_graph/github-mark.png"},
	{"https://stackoverflow.com", "Stack Overflow", "Pair programming with AI", "https://cdn.sstatic.net/Sites/stackoverflow/Img/apple-touch-icon@2.png"},
	{"https://www.youtube.com", "YouTube", "Enjoy the videos and music you love", "https://www.youtube.com/img/desktop/yt_1200.png"},
	{"https://twitter.com", "X (Twitter)", "The heartbeat of the world", "https://abs.twimg.com/responsive-web/client-web/icon-ios.77d25eba.png"},
	{"https://www.linkedin.com", "LinkedIn", "Manage your professional identity", "https://static.licdn.com/scds/common/u/images/logos/favicons/v1/default.png"},
	{"https://www.reddit.com", "Reddit", "Dive into anything", "https://www.redditstatic.com/desktop2x/img/favicon/android-icon-192x192.png"},
	{"https://medium.com", "Medium", "Where good ideas find you", "https://miro.medium.com/max/1200/1*jfdwtvU6V6g99q3G7gq7dQ.png"},
	{"https://www.figma.com", "Figma", "The collaborative interface design tool", "https://cdn.sanity.io/images/599r6htc/localized/46a76c802176eb17b04e12108f7e7c0377e09228-1024x1024.png"},
	{"https://www.notion.so", "Notion", "Your connected workspace for wiki, docs & projects", "https://www.notion.so/images/meta/default.png"},
	{"https://vercel.com", "Vercel", "Develop. Preview. Ship.", "https://vercel.com/api/www/avatar?u=vercel&s=180"},
	{"https://nextjs.org", "Next.js", "The React Framework for the Web", "https://nextjs.org/static/twitter-cards/home.jpg"},
	{"https://tailwindcss.com", "Tailwind CSS", "Rapidly build modern websites without ever leaving your HTML", "https://tailwindcss.com/_next/static/media/twitter-square.daf77586b35e90319725e742f6e069f9.jpg"},
	{"https://react.dev", "React", "The library for web and native user interfaces", "https://react.dev/images/og-home.png"},
	{"https://go.dev", "Go", "Build fast, reliable, and efficient software at scale", "https://go.dev/images/go-logo-white.svg"},
	{"https://www.docker.com", "Docker", "Develop shipping containers", "https://www.docker.com/wp-content/uploads/2022/03/social-docker-logo.png"},
	{"https://aws.amazon.com", "AWS", "Amazon Web Services", "https://a0.awsstatic.com/libra-css/images/logos/aws_logo_smile_1200x630.png"},
	{"https://cloud.google.com", "Google Cloud", "Cloud Computing Services", "https://cloud.google.com/_static/cloud/images/social-icon-google-cloud-1200-630.png"},
	{"https://www.producthunt.com", "Product Hunt", "The best new products in tech", "https://ph-static.imgix.net/ph-ios-icon.png"},
	{"https://dribbble.com", "Dribbble", "Discover the world's top designers & creatives", "https://cdn.dribbble.com/assets/dribbble-ball-mark-2bd45f09c2fb58dbbfb44571e5f1e073.svg"},
	{"https://www.spotify.com", "Spotify", "Listen to music, podcasts and more", "https://developer.spotify.com/assets/branding-guidelines/icon1@2x.png"},
	{"https://netflix.com", "Netflix", "Watch TV shows and movies online", "https://images.ctfassets.net/y2ske730sjqp/1aONibCke6niZhgPxuiilC/2c401b05a07288746ddf3bd3943fbc76/BrandAssets_Logos_01-Wordmark.jpg"},
	{"https://www.amazon.com", "Amazon", "Spend less. Smile More.", "https://m.media-amazon.com/images/G/01/gno/sprites/nav-sprite-global-2x._CB405765698_.png"},
	{"https://www.typescriptlang.org", "TypeScript", "JavaScript With Syntax For Types", "https://www.typescriptlang.org/icons/icon-512x512.png"},
	{"https://www.rust-lang.org", "Rust", "A language empowering everyone to build reliable software", "https://www.rust-lang.org/static/images/rust-social-wide.jpg"},
	{"https://kubernetes.io", "Kubernetes", "Production-Grade Container Orchestration", "https://kubernetes.io/images/kubernetes-horizontal-color.png"},
	{"https://www.postgresql.org", "PostgreSQL", "The World's Most Advanced Open Source Relational Database", "https://www.postgresql.org/media/img/about/press/elephant.png"},
	{"https://www.mongodb.com", "MongoDB", "The developer data platform", "https://webimages.mongodb.com/_com_assets/cms/kuyjf3vea2hg34taa-horizontal_default_slate_blue.svg"},
	{"https://www.npmjs.com", "npm", "Build amazing things", "https://static.npmjs.com/338e4905a2684ca96e08c7780fc68412.png"},
	{"https://code.visualstudio.com", "VS Code", "Code editing. Redefined.", "https://code.visualstudio.com/opengraphimg/opengraph-home.png"},
	{"https://www.apple.com", "Apple", "Newsroom - Apple", "https://www.apple.com/ac/structured-data/images/knowledge_graph_logo.png"},
	{"https://developer.mozilla.org", "MDN Web Docs", "Resources for developers, by developers", "https://developer.mozilla.org/mdn-social-share.png"},
	{"https://www.cloudflare.com", "Cloudflare", "The Web Performance & Security Company", "https://cf-assets.www.cloudflare.com/slt3lc6tev37/CHOl0sUhrumCxOXfRotDt/8c777e7613ef4978f3a3a99b7281ea6d/CF_logo_stacked_darkmode.svg"},
	{"https://stripe.com", "Stripe", "Financial infrastructure for the internet", "https://images.ctfassets.net/fzn2n1nzq965/HTTOloNPhisV9P4hlMPNA/cacf1bb88b9fc492dfad34378d844280/Stripe_icon_-_square.svg"},
	{"https://www.canva.com", "Canva", "Free Design Tool: Presentations, Video, Social Media", "https://static.canva.com/static/images/og-image.jpg"},
	{"https://www.atlassian.com", "Atlassian", "Tools for teams, from startup to enterprise", "https://wac-cdn.atlassian.com/dam/jcr:1f11bc60-9e8d-4e41-be4e-6cd0edc3282f/Atlassian-horizontal-blue-rgb.svg"},
	{"https://www.openai.com", "OpenAI", "Creating safe AGI that benefits all of humanity", "https://openai.com/content/images/2022/05/openai-avatar.png"},
	{"https://www.tesla.com", "Tesla", "Electric Cars, Solar & Clean Energy", "https://www.tesla.com/themes/custom/flavor/icon/social/tesla.png"},
	{"https://www.udemy.com", "Udemy", "Online Courses - Learn Anything", "https://www.udemy.com/staticx/udemy/images/v7/logo-udemy.svg"},
	{"https://www.coursera.org", "Coursera", "Build Skills with Online Courses from Top Institutions", "https://coursera.org/og_image.jpg"},
	{"https://www.twitch.tv", "Twitch", "Interactive live streaming service for content", "https://brand.twitch.tv/assets/images/black.png"},
	{"https://discord.com", "Discord", "Your Place to Talk and Hang Out", "https://discord.com/assets/847541504914fd33810e70a0ea73177e.ico"},
	{"https://www.slack.com", "Slack", "Where work happens", "https://a.slack-edge.com/80588/marketing/img/meta/slack_hash_256.png"},
	{"https://www.trello.com", "Trello", "Manage Your Team's Projects From Anywhere", "https://d2k1ftgv7pobq7.cloudfront.net/meta/p/res/images/trello-header-logos/167dc7b9900a5b241b15ba21f8571571/trello-logo-blue.svg"},
	{"https://www.behance.net", "Behance", "Best of Behance", "https://a5.behance.net/images/brand/apple-touch-icon.png"},
	{"https://www.pixiv.net", "pixiv", "Illustrations, manga, and fan fiction", "https://www.pixiv.net/favicon.ico"},
	{"https://www.wikipedia.org", "Wikipedia", "The free encyclopedia", "https://en.wikipedia.org/static/images/project-logos/enwiki.png"},
}

var browsers = []string{"Chrome 120.0", "Firefox 121.0", "Safari 17.2", "Edge 120.0", "Brave 1.61", "Opera 105.0", "Chrome 119.0", "Firefox 120.0", "Safari 16.6"}
var osNames = []string{"Windows 11", "macOS Sonoma", "Ubuntu 24.04", "iOS 17", "Android 14", "Windows 10", "macOS Ventura", "iOS 16", "Android 13", "ChromeOS"}
var devices = []string{"Desktop", "Mobile", "Mobile", "Desktop", "Desktop", "Mobile", "Tablet", "Desktop", "Mobile", "Desktop"}
var countries = []string{"United States", "India", "United Kingdom", "Germany", "Japan", "Canada", "Brazil", "Australia", "France", "Netherlands", "Singapore", "South Korea"}
var cities = []string{"San Francisco", "Mumbai", "London", "Berlin", "Tokyo", "Toronto", "São Paulo", "Sydney", "Paris", "Amsterdam", "Singapore", "Seoul", "New York", "Bangalore", "Chicago"}
var referrers = []string{"https://google.com", "https://twitter.com", "https://linkedin.com", "Direct", "https://reddit.com", "https://facebook.com", "https://producthunt.com", "https://news.ycombinator.com", "https://medium.com", ""}
var ips = []string{"192.168.1.1", "10.0.0.1", "172.16.0.1", "8.8.8.8", "1.1.1.1", "203.0.113.1", "198.51.100.1", "100.0.0.1"}

func SeedData(c *gin.Context) {
	// Create test user if not exists
	var user models.User
	err := config.DB.Where("email = ?", "testuser@jugaadlink.com").First(&user).Error
	if err != nil {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("test123456"), bcrypt.DefaultCost)
		user = models.User{
			Email:    "testuser@jugaadlink.com",
			Name:     "Test User",
			Password: string(hashed),
		}
		config.DB.Create(&user)
	}

	urlCount := 150
	createdURLs := make([]models.URL, 0, urlCount)

	for i := 0; i < urlCount; i++ {
		sample := sampleURLs[i%len(sampleURLs)]
		shortCode := util.GenerateShortCode()

		clickCount := rand.Intn(200) + 1
		createdDaysAgo := rand.Intn(60) + 1
		createdAt := time.Now().AddDate(0, 0, -createdDaysAgo)

		url := models.URL{
			OriginalURL: sample.URL,
			ShortCode:   shortCode,
			UserID:      &user.ID,
			Clicks:      clickCount,
			Title:       sample.Title,
			Description: sample.Desc,
			Image:       sample.Image,
		}
		url.CreatedAt = createdAt

		if err := config.DB.Create(&url).Error; err != nil {
			continue
		}
		createdURLs = append(createdURLs, url)
	}

	// Generate clicks for each URL
	totalClicks := 0
	for _, url := range createdURLs {
		numClicks := url.Clicks
		for j := 0; j < numClicks; j++ {
			clickDaysAgo := rand.Intn(30)
			clickTime := time.Now().AddDate(0, 0, -clickDaysAgo).Add(
				time.Duration(rand.Intn(24)) * time.Hour,
			).Add(
				time.Duration(rand.Intn(60)) * time.Minute,
			)

			click := models.Click{
				URLID:     url.ID,
				IP:        ips[rand.Intn(len(ips))],
				UserAgent: "Mozilla/5.0 (compatible; SeedBot/1.0)",
				Browser:   browsers[rand.Intn(len(browsers))],
				OS:        osNames[rand.Intn(len(osNames))],
				Device:    devices[rand.Intn(len(devices))],
				Country:   countries[rand.Intn(len(countries))],
				City:      cities[rand.Intn(len(cities))],
				Referer:   referrers[rand.Intn(len(referrers))],
				Timestamp: clickTime,
			}

			config.DB.Create(&click)
			totalClicks++
		}
	}

	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"message":      "Seed data created successfully",
		"urls_created":  len(createdURLs),
		"clicks_created": totalClicks,
		"test_user": gin.H{
			"email":    "testuser@jugaadlink.com",
			"password": "test123456",
		},
	}))
}

func ResetSeedData(c *gin.Context) {
	// Delete all clicks
	config.DB.Exec("DELETE FROM clicks")
	// Delete all URLs
	config.DB.Exec("DELETE FROM urls")
	// Delete test user
	config.DB.Exec("DELETE FROM users WHERE email = 'testuser@jugaadlink.com'")

	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"message": "All seed data cleared",
	}))
}
