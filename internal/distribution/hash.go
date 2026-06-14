package distribution

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const skillHashHeader = "context-skill-hash-v1\x00"

const (
	hashDirectoryRecord byte = 0x01
	hashFileRecord      byte = 0x02
)

type hashEntry struct {
	relativePath string
	mode         fs.FileMode
	isDirectory  bool
}

// HashSkill はSkillディレクトリを安全に走査して決定的なハッシュを返します。
//
//nolint:gocognit,cyclop // 走査中の安全性検証と正規化レコード生成を同じ境界で保証します。
func HashSkill(root string) (string, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return "", newError("hash inspect", ErrIO, err)
	}
	if err := validateHashEntry(rootInfo, true); err != nil {
		return "", err
	}

	entries := []hashEntry{{relativePath: ".", mode: rootInfo.Mode(), isDirectory: true}}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return fmt.Errorf("エントリ情報取得: %w", infoErr)
		}
		if validateErr := validateHashEntry(info, entry.IsDir()); validateErr != nil {
			return validateErr
		}
		relative, relativeErr := filepath.Rel(root, path)
		if relativeErr != nil {
			return fmt.Errorf("相対パス計算: %w", relativeErr)
		}
		relative = filepath.ToSlash(relative)
		if relative == "" || relative == "." || relative == ".." ||
			strings.HasPrefix(relative, "../") || filepath.IsAbs(relative) {
			return ErrUnsafePath
		}
		entries = append(entries, hashEntry{
			relativePath: relative,
			mode:         info.Mode(),
			isDirectory:  entry.IsDir(),
		})
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrSymlink), errors.Is(err, ErrFileType), errors.Is(err, ErrUnsafePath):
			return "", newError("hash walk", ErrStructure, err)
		default:
			return "", newError("hash walk", ErrIO, err)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].relativePath < entries[j].relativePath
	})

	hash := sha256.New()
	if _, err := io.WriteString(hash, skillHashHeader); err != nil {
		return "", newError("hash header", ErrIO, err)
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if _, exists := seen[entry.relativePath]; exists {
			return "", newError("hash path", ErrUnsafePath, nil)
		}
		seen[entry.relativePath] = struct{}{}
		if err := writeHashEntry(hash, root, entry); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validateHashEntry(info fs.FileInfo, directory bool) error {
	if info.Mode()&fs.ModeSymlink != 0 {
		return newError("hash inspect", ErrSymlink, nil)
	}
	if directory {
		if !info.IsDir() {
			return newError("hash inspect", ErrFileType, nil)
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return newError("hash inspect", ErrFileType, nil)
	}
	return nil
}

func writeHashEntry(writer io.Writer, root string, entry hashEntry) error {
	kind := hashFileRecord
	if entry.isDirectory {
		kind = hashDirectoryRecord
	}
	if err := binary.Write(writer, binary.BigEndian, kind); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	pathBytes := []byte(entry.relativePath)
	if err := binary.Write(writer, binary.BigEndian, uint64(len(pathBytes))); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	if _, err := writer.Write(pathBytes); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	if err := binary.Write(writer, binary.BigEndian, uint32(entry.mode.Perm())); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	if entry.isDirectory {
		if err := binary.Write(writer, binary.BigEndian, uint64(0)); err != nil {
			return newError("hash encode", ErrIO, err)
		}
		return nil
	}

	path := filepath.Join(root, filepath.FromSlash(entry.relativePath))
	file, err := os.Open(path) // #nosec G304 -- 直前にLstat済みのSkill配下通常ファイルだけを開きます。
	if err != nil {
		return newError("hash open", ErrIO, err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return newError("hash stat", ErrIO, err)
	}
	if !info.Mode().IsRegular() {
		return newError("hash inspect", ErrFileType, nil)
	}
	if info.Size() < 0 {
		return newError("hash inspect", ErrFileType, nil)
	}
	// #nosec G115 -- 負値を直前で拒否済みです。
	if err := binary.Write(writer, binary.BigEndian, uint64(info.Size())); err != nil {
		return newError("hash encode", ErrIO, err)
	}
	buffered := bufio.NewReader(file)
	if _, err := io.Copy(writer, buffered); err != nil {
		return newError("hash content", ErrIO, fmt.Errorf("ファイル内容読み込み: %w", err))
	}
	return nil
}
