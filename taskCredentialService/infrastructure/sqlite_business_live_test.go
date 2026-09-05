package infrastructure

import (
	"testing"
)

func TestLiveFetchRepoIdentitiesAndProvider(t *testing.T) {
	root := "/tmp/ram-work"
	taskDB, projectDB, err := OpenBusinessDBs(root)
	if err != nil {
		t.Skipf("business DBs unavailable: %v", err)
	}
	repo := NewSQLiteBusinessRepository(taskDB, projectDB)
	ids, err := repo.FetchRepoIdentities("task_12675381068363715869", "")
	if err != nil {
		t.Skipf("business DB schema/data unavailable: %v", err)
	}
	t.Logf("identities=%+v", ids)
	for _, id := range ids {
		if id.UserID <= 0 {
			t.Errorf("user id missing for %s identity=%s", id.RepoURL, id.GitIdentityID)
		}
	}
	repos, err := repo.FetchTaskRepos("task_12675381068363715869", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("repos=%+v", repos)
	resolver, err := LoadProviderConfigs(root)
	if err != nil {
		t.Fatal(err)
	}
	client := NewGitoauthHTTPClient("http://127.0.0.1:8002", 5, resolver, "")
	for _, r := range repos {
		for _, u := range r.RepoURLs {
			p := client.ResolveProvider(u)
			t.Logf("url=%s provider=%q", u, p)
			if p == "" {
				t.Errorf("provider empty for %s", u)
			}
		}
	}
}
