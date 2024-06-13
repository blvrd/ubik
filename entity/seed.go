package entity

import "time"

func Seed() map[string]any {
  seedData := make(map[string]any)

  seedData["issues"] = []Issue{
    {
      Id: "12345",
      shortcode: "ABC123",
      Author: "garrett@blvrd.co",
      Title: "git ref storage prototype",
      Description: "",
      CreatedAt: time.Now().UTC(),
      UpdatedAt: time.Now().UTC(),
      Comments: []Comment{
        {
          Author: "garrett@blvrd.co",
          Body: "Lorem ipsum dolor sit amet",
        },
        {
          Author: "harsha@example.com",
          Body: "Lorem ipsum dolor sit amet",
        },
        {
          Author: "codestyle@bot",
          Body: "I smell a code smell.",
        },
      },
    },
    {
      Id: "54321",
      shortcode: "123CBA",
      Author: "garrett@blvrd.co",
      Title: "Some other issue",
      Description: "asdf",
      CreatedAt: time.Now().UTC(),
      UpdatedAt: time.Now().UTC(),
      Comments: []Comment{
        {
          Author: "garrett@blvrd.co",
          Body: "Lorem ipsum dolor sit amet",
        },
        {
          Author: "harsha@example.com",
          Body: "Lorem ipsum dolor sit amet",
        },
        {
          Author: "codestyle@bot",
          Body: "I smell a code smell.",
        },
      },
    },
  }

  var issuePointers []*Issue

  for _, issue := range seedData["issues"].([]Issue) {
    issue := issue
    issuePointers = append(issuePointers, &issue)
  }

  seedData["projects"] = []Project{
    {
      id: "54321",
      Title: "Persisting data",
      Description: "How are we going to persist data in Ubik?",
      Issues: issuePointers,
    },
  }

  return seedData
}
