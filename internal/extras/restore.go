package extras

import (
	"fmt"
	"strings"
	"time"

	"fpsforge/internal/system"
)

// CreateRestorePoint intenta crear un punto de restauración del sistema.
// Windows limita uno cada 24 h por defecto; se levanta ese límite
// temporalmente para que siempre exista un punto justo antes de cambiar nada.
func CreateRestorePoint(sys system.System, desc string) error {
	desc = strings.NewReplacer("'", "", `"`, "", "`", "", "$", "").Replace(desc)
	script := `$k = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\SystemRestore'
$old = $null
try { $old = (Get-ItemProperty -LiteralPath $k -Name SystemRestorePointCreationFrequency -ErrorAction Stop).SystemRestorePointCreationFrequency } catch {}
try {
  New-ItemProperty -LiteralPath $k -Name SystemRestorePointCreationFrequency -Value 0 -PropertyType DWord -Force | Out-Null
  Enable-ComputerRestore -Drive ($env:SystemDrive + '\') -ErrorAction SilentlyContinue
  Checkpoint-Computer -Description '` + desc + `' -RestorePointType MODIFY_SETTINGS -ErrorAction Stop
} finally {
  if ($null -ne $old) { Set-ItemProperty -LiteralPath $k -Name SystemRestorePointCreationFrequency -Value $old } 
  else { Remove-ItemProperty -LiteralPath $k -Name SystemRestorePointCreationFrequency -ErrorAction SilentlyContinue }
}`
	if _, err := sys.PowerShell(script, 3*time.Minute); err != nil {
		return fmt.Errorf("no se pudo crear el punto de restauración (¿protección del sistema desactivada?): %w", err)
	}
	return nil
}
