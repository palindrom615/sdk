package sdkman

import (
	"fmt"
	"github.com/scylladb/go-set/strset"
	"github.com/urfave/cli/v2"
	"path"
	"strings"
)

func Install(c *cli.Context) error {
	target := Sdk{c.Args().Get(0), c.Args().Get(1)}
	folder := c.Args().Get(2)

	reg := c.String("registry")
	root := c.String("directory")

	_ = Update(c)

	MkdirIfNotExist(root)
	if err := checkValidCand(root, target.Candidate); err != nil {
		return err
	}
	if target.Version == "" {
		if defaultSdk, err := defaultSdk(reg, root, target.Candidate); err != nil {
			return err
		} else {
			target = defaultSdk
		}
	}

	if target.IsInstalled(root) {
		return ErrVerExists
	}
	if err := target.checkValidVer(reg, root, folder); err != nil {
		return err
	}

	archiveReady := make(chan bool)
	installReady := make(chan bool)
	go target.Unarchive(root, archiveReady, installReady)
	if target.IsArchived(root) {
		archiveReady <- true
	} else {
		s, err, t := getDownload(reg, target)
		if err != nil {
			archiveReady <- false
			return err
		}
		archive := Archive{target, t}
		go archive.Save(s, root, archiveReady)
	}
	if <-installReady == false {
		return ErrVerInsFail
	}
	return target.Use(root)
}

func Use(c *cli.Context) error {
	candidate := c.Args().Get(0)
	version := c.Args().Get(1)
	sdk := Sdk{candidate, version}
	root := c.String("directory")
	if err := checkValidCand(root, candidate); err != nil {
		return err
	}
	if !sdk.IsInstalled(root) {
		return ErrVerNotIns
	}
	return sdk.Use(root)
}

func Current(c *cli.Context) error {
	candidate := c.Args().Get(0)
	root := c.String("directory")
	if candidate == "" {
		sdks := CurrentSdks(root)
		if len(sdks) == 0 {
			return ErrNoCurrCands
		}
		for _, sdk := range sdks {
			fmt.Printf("%s@%s\n", sdk.Candidate, sdk.Version)
		}
	} else {
		sdk, err := CurrentSdk(root, candidate)
		if err == nil {
			fmt.Println(sdk.Candidate + "@" + sdk.Version)
		} else {
			return ErrNoCurrSdk(candidate)
		}
	}
	return nil
}

type envVar struct {
	name string
	val  string
}

func Export(c *cli.Context) error {
	shell := c.Args().Get(0)
	root := c.String("directory")
	sdks := CurrentSdks(c.String("directory"))
	if len(sdks) == 0 {
		fmt.Println("")
		return nil
	}
	paths := []string{}
	homes := []envVar{}
	for _, sdk := range sdks {
		candHome := path.Join(root, "candidates", sdk.Candidate, "current")
		paths = append(paths, path.Join(candHome, "bin"))
		homes = append(homes, envVar{fmt.Sprintf("%s_HOME", strings.ToUpper(sdk.Candidate)), candHome})
	}

	if shell == "bash" || shell == "" {
		evalBash(paths, homes)
	}
	return nil
}

func List(c *cli.Context) error {
	candidate := c.Args().Get(0)
	reg := c.String("registry")
	root := c.String("directory")

	if candidate == "" {
		list, err := getList(reg)
		if err == nil {
			pager(list)
		}
		return err
	} else {
		if err := checkValidCand(root, candidate); err != nil {
			return err
		}
		ins := InstalledSdks(root, candidate)
		curr, _ := CurrentSdk(root, candidate)
		list, err := getVersionsList(reg, curr, ins)
		pager(list)
		return err
	}
}

func Update(c *cli.Context) error {
	reg := c.String("registry")
	root := c.String("directory")
	freshCsv, netErr := getAll(reg)
	if netErr != nil {
		return ErrNotOnline
	}
	fresh := strset.New(freshCsv...)
	cachedCsv := getCandidates(root)
	cached := strset.New(cachedCsv...)

	added := strset.Difference(fresh, cached)
	obsoleted := strset.Difference(cached, fresh)

	if added.Size() == 0 && obsoleted.Size() == 0 {
		fmt.Println("No new candidates found at this time.")
	} else {
		fmt.Println("Adding new candidates: " + strings.Join(added.List(), ", "))
		fmt.Println("Removing obsolete candidates: " + strings.Join(obsoleted.List(), ", "))
		_ = setCandidates(root, freshCsv)
	}
	return nil
}

func evalBash(paths []string, envVars []envVar) {
	fmt.Println("export PATH=" + strings.Join(paths, ":") + ":$PATH")
	for _, v := range envVars {
		fmt.Println("export " + v.name + "=" + v.val)
	}
}

