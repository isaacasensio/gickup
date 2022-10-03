package onedev

import (
	"fmt"

	"github.com/cooperspencer/gickup/types"
	"github.com/cooperspencer/onedev"
	"github.com/rs/zerolog/log"
)

func Get(conf *types.Conf) []types.Repo {
	repos := []types.Repo{}

	for _, repo := range conf.Source.OneDev {
		if repo.URL == "" {
			repo.URL = "https://code.onedev.io/"
		}

		include := types.GetMap(repo.Include)
		exclude := types.GetMap(repo.Exclude)
		excludeorgs := types.GetMap(repo.ExcludeOrgs)

		log.Info().
			Str("stage", "onedev").
			Str("url", repo.URL).
			Msgf("grabbing repositories from %s", repo.User)

		if repo.Password == "" && repo.Token != "" {
			repo.Password = repo.Token
		}

		client := &onedev.Client{}

		if repo.Password != "" {
			client = onedev.NewClient(repo.URL, repo.Username, repo.Password)
		} else {
			client = onedev.NewClient(repo.URL, "", "")
		}

		query := onedev.ProjectQueryOptions{
			Query:  "",
			Offset: 0,
			Count:  100,
		}

		user := onedev.User{}

		if repo.User == "" {
			u, err := client.GetMe()
			if err != nil {
				log.Fatal().
					Str("stage", "onedev").
					Str("url", repo.URL).
					Msg("can't find user")
				break
			}
			user = u
			repo.User = user.Name
		}

		if repo.User != "" {
			query.Query = fmt.Sprintf("owned by \"%s\"", repo.User)
		}

		userrepos, err := client.GetProjects(&query)
		if err != nil {
			log.Fatal().
				Str("stage", "onedev").
				Str("url", repo.URL).
				Msg(err.Error())
		}

		for _, r := range userrepos {
			if len(repo.Include) > 0 {
				if !include[r.Name] {
					continue
				}
				if exclude[r.Name] {
					continue
				}
			}

			urls, err := client.GetCloneUrl(r.ID)
			if err != nil {
				log.Fatal().
					Str("stage", "onedev").
					Str("url", repo.URL).
					Msg("couldn't get clone urls")
			}

			defaultbranch, err := client.GetDefaultBranch(r.ID)
			if err != nil {
				fmt.Println(err)
				log.Fatal().
					Str("stage", "onedev").
					Str("url", repo.URL).
					Msgf("couldn't get default branch for %s", r.Name)
				defaultbranch = "main"
			}

			repos = append(repos, types.Repo{
				Name:          r.Name,
				URL:           urls.HTTP,
				SSHURL:        urls.SSH,
				Token:         repo.Token,
				Defaultbranch: defaultbranch,
				Origin:        repo,
				Owner:         repo.User,
				Hoster:        types.GetHost(repo.URL),
			})
		}

		if repo.Username != "" && repo.Password != "" && len(repo.IncludeOrgs) == 0 && user.Name != "" {
			memberships, err := client.GetUserMemberships(user.ID)
			if err != nil {
				log.Error().
					Str("stage", "onedev").
					Str("url", repo.URL).
					Msgf("couldn't get memberships for %s", user.Name)
			}

			for _, membership := range memberships {
				group, err := client.GetGroup(membership.GroupID)
				if err != nil {
					log.Error().
						Str("stage", "onedev").
						Str("url", repo.URL).
						Msgf("couldn't get group with id %d", membership.GroupID)
				}
				if !excludeorgs[group.Name] {
					repo.IncludeOrgs = append(repo.IncludeOrgs, group.Name)
				}
			}
		}

		fmt.Println(repo.IncludeOrgs)

		if len(repo.IncludeOrgs) > 0 {
			for _, org := range repo.IncludeOrgs {
				query.Query = fmt.Sprintf("children of \"%s\"", org)

				orgrepos, err := client.GetProjects(&query)
				if err != nil {
					log.Fatal().
						Str("stage", "onedev").
						Str("url", repo.URL).
						Msg(err.Error())
				}

				for _, r := range orgrepos {
					urls, err := client.GetCloneUrl(r.ID)
					if err != nil {
						log.Fatal().
							Str("stage", "onedev").
							Str("url", repo.URL).
							Msg("couldn't get clone urls")
					}

					defaultbranch, err := client.GetDefaultBranch(r.ID)
					if err != nil {
						log.Fatal().
							Str("stage", "onedev").
							Str("url", repo.URL).
							Msgf("couldn't get default branch for %s", r.Name)
						defaultbranch = "main"
					}

					repos = append(repos, types.Repo{
						Name:          r.Name,
						URL:           urls.HTTP,
						SSHURL:        urls.SSH,
						Token:         repo.Token,
						Defaultbranch: defaultbranch,
						Origin:        repo,
						Owner:         org,
						Hoster:        types.GetHost(repo.URL),
					})
				}
			}
		}
	}

	return repos
}