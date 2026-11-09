package agent

import (
	"context"
	"encoding/json"
)

func (d Dialog) desktop(ctx context.Context, title, prompt string, secret bool) (string, error) {
	data, _ := json.Marshal(map[string]any{"title": title, "prompt": prompt, "secret": secret})
	const script = `ObjC.import('Foundation'); const app=Application.currentApplication(); app.includeStandardAdditions=true;
 const data=$.NSFileHandle.fileHandleWithStandardInput.readDataToEndOfFile;
 const r=JSON.parse(ObjC.unwrap($.NSString.alloc.initWithDataEncoding(data,$.NSUTF8StringEncoding)));
 try {const opts={withTitle:r.title,buttons:['Deny','Allow'],defaultButton:'Deny',cancelButton:'Deny'};
 if(r.secret){opts.defaultAnswer='';opts.hiddenAnswer=true;opts.defaultButton='Allow';}
 const result=app.displayDialog(r.prompt,opts); if(r.secret) result.textReturned; else '';}
 catch(e){$.exit(1);}`
	return d.run(ctx, "/usr/bin/osascript", []string{"-l", "JavaScript", "-e", script}, string(data))
}
