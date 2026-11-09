package agent

import (
	"context"
	"encoding/json"
)

func (d Dialog) desktopSelect(ctx context.Context, title, prompt string, labels []string) (int, error) {
	data, _ := json.Marshal(map[string]any{"title": title, "prompt": prompt, "labels": labels})
	const script = `ObjC.import('Foundation'); const app=Application.currentApplication(); app.includeStandardAdditions=true;
 const data=$.NSFileHandle.fileHandleWithStandardInput.readDataToEndOfFile;
 const r=JSON.parse(ObjC.unwrap($.NSString.alloc.initWithDataEncoding(data,$.NSUTF8StringEncoding)));
 try { const result=app.chooseFromList(r.labels,{withTitle:r.title,withPrompt:r.prompt,okButtonName:'Select',cancelButtonName:'Cancel',multipleSelectionsAllowed:false});
 if(result===false) $.exit(1); result[0]; } catch(e) { $.exit(1); }`
	value, err := d.run(ctx, "/usr/bin/osascript", []string{"-l", "JavaScript", "-e", script}, string(data))
	if err != nil {
		return -1, err
	}
	return selectedLabel(value, labels)
}
