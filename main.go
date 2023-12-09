package main

import "database/sql"
import _ "github.com/glebarez/go-sqlite"
import "debug/elf"
import "fmt"
import "log"
import "os"
import "strings"
//import "github.com/nesv/rpm/spec"
import "github.com/xor-gate/debpkg"

func main() {
	if !file_exists("spec/header") {
		log.Fatal("spec/header not found")
	}

	// Read the spec file
//	specfile, readerr := os.Open("spec/header")
//	if readerr != nil {
//		log.Fatal(readerr)
//	}
//	defer specfile.Close()

//	specdata, specerr := spec.ParseReader(specfile)
	specdata, specerr := parseSpecFile("spec/header")
	if specerr != nil {
		log.Fatal(specerr)
	}

	files := get_filelist("./helper.db")

	// New deb package
	deb := debpkg.New()

	// Apply settings from spec file to new deb package
//	deb.SetName(specdata.Name())
//	deb.SetVersion(specdata.Version())
	deb.SetName(specdata.Name)
	deb.SetVersion(specdata.Version)
	deb.SetMaintainer(specdata.Packager)
	deb.SetMaintainerEmail(specdata.PackagerEmail)
	deb.SetShortDescription(specdata.Summary)

	// Iterate over files
	arch := make(map[string]int)
	for _, file := range files {
		fmt.Printf("Adding %s\n", "root/"+file)
		arch[binary_deb_arch("root/"+file)]++
		deb.AddFile("root/"+file, strings.TrimLeft(file, "/"))
	}

	delete(arch, "unknown")
	if len(arch) > 1 {
		log.Fatal("Found binaries for different architectures. This is unsupported")
	}
	if len(arch) == 0 {
		// If no arch was detected, we set it to `all`
		arch["all"] = 1
	}

	// This is a loop, but there is only one element, so it should be safe
	var debarch string
	for k, _ := range arch {
		debarch = k
	}
	deb.SetArchitecture(debarch)

	debfile := fmt.Sprintf("%s_%s_%s.deb", specdata.Name, specdata.Version, debarch)
	if file_exists(debfile) {
		log.Fatalf("%s already exists, not overwriting it\n", debfile)
	}

	writeerr := deb.Write(debfile)
	if writeerr != nil { log.Fatal(writeerr) }

	fmt.Printf("Wrote %s\n", debfile)

	defer deb.Close()
}

func binary_deb_arch (name string) (string) {
	file, openerr := elf.Open(name)
	if openerr != nil { return "unknown" }
	defer file.Close()

	switch file.FileHeader.Machine.String() {
	case "EM_386":
		return "i386"
	case "EM_X86_64":
		return "amd64"
	case "EM_ARM":
		return "arm"
	case "EM_AARCH64":
		return "arm64"
	default:
		return "unknown"
	}
}

func get_filelist (dbfile string) ([]string) {
	var filelist []string

	// sqlite will create the file, if it does not exist, so we need to check for it explicitly
	if !file_exists(dbfile) {
		log.Fatalf("%s not found\n", dbfile)
	}

	db, openerr := sql.Open("sqlite", dbfile)
	if openerr != nil {
		log.Fatal(openerr)
	}
	defer db.Close()

	row, qerr := db.Query("SELECT path FROM package_file WHERE NOT excluded")
	if qerr != nil {
		log.Fatal(qerr)
	}
	defer row.Close()
	
	for row.Next() {
		var path string
		row.Scan(&path)
		filelist = append(filelist, path)
	}

	return filelist
}

func file_exists (path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
