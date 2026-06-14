package distribution

const skillHashHeader = "context-skill-hash-v1\x00"

const (
	hashDirectoryRecord byte = 0x01
	hashFileRecord      byte = 0x02
)

// HashSkill はSkillディレクトリをFD起点で安全に走査して決定的なハッシュを返します。
func HashSkill(root string) (string, error) {
	return newSafeTree().hash(root)
}
