package agent

import (
	"context"
	"encoding/json"
)

func (d Dialog) desktopSelect(ctx context.Context, title, prompt string, labels []string) (int, error) {
	data, _ := json.Marshal(map[string]any{"title": title, "prompt": prompt, "labels": labels})
	const script = `Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing;
 $r=[Console]::In.ReadToEnd() | ConvertFrom-Json;
 $f=New-Object System.Windows.Forms.Form; $f.Text=$r.title; $f.Size=New-Object System.Drawing.Size(800,400); $f.StartPosition='CenterScreen';
 $l=New-Object System.Windows.Forms.Label; $l.Text=$r.prompt; $l.SetBounds(15,15,750,35); $f.Controls.Add($l);
 $list=New-Object System.Windows.Forms.ListBox; $list.SetBounds(15,55,750,245); foreach($v in $r.labels){[void]$list.Items.Add($v)}; $f.Controls.Add($list);
 $ok=New-Object System.Windows.Forms.Button; $ok.Text='Select'; $ok.SetBounds(580,315,80,30); $ok.DialogResult='OK'; $f.Controls.Add($ok);
 $cancel=New-Object System.Windows.Forms.Button; $cancel.Text='Cancel'; $cancel.SetBounds(675,315,80,30); $cancel.DialogResult='Cancel'; $f.Controls.Add($cancel); $f.AcceptButton=$ok; $f.CancelButton=$cancel;
 if($f.ShowDialog() -ne 'OK' -or $list.SelectedIndex -lt 0){exit 1}; [Console]::Out.Write($list.SelectedItem); $f.Dispose();`
	value, err := d.run(ctx, "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-STA", "-Command", script}, string(data))
	if err != nil {
		return -1, err
	}
	return selectedLabel(value, labels)
}
