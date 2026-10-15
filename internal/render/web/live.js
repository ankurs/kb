(function(){'use strict';
const config=JSON.parse(document.getElementById('viewer-config').textContent);
const mounts=config.mounts||[],trustHTML=!!config.trustHTML;
mermaid.initialize({startOnLoad:false,theme:'dark',securityLevel:'strict'});
window.$docsify={
  name:config.name,homepage:'README.md',loadSidebar:true,
  alias:{'/.*/_sidebar.md':'/_sidebar.md'},relativePath:true,subMaxLevel:3,
  markdown:{renderer:{code:function(code,lang){
    if(lang==='mermaid'){
      let safe=trustHTML?code:code.replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
      return '<div class="mermaid">'+safe+'</div>'
    }
    return this.origin.code.apply(this,arguments)
  }}},
  plugins:[function(hook,vm){
    hook.beforeEach(function(md){
      let mount=mounts.find(m=>vm.route.path.startsWith('/'+m+'/'));
      if(!mount)return md;
      let fence=false;
      return md.split('\n').map(line=>{
        let t=line.trim();if(t.startsWith('~~~')||t.charCodeAt(0)===96)fence=!fence;
        if(fence)return line;
        return line.replace(/\]\(\/([^)]+)/g,(all,dest)=>dest.startsWith(mount+'/')?all:'](/'+mount+'/'+dest)
      }).join('\n')
    });
    hook.afterEach(function(html){return trustHTML?html:DOMPurify.sanitize(html)});
    hook.doneEach(function(){
      window.kbViewerUI?.setActive();
      KBUI.externalLinks(document.querySelector('.markdown-section'),config.externalLinksNewTab);
      mermaid.run({querySelector:'.mermaid'}).catch(console.error)
    })
  }]
};
fetch('/_kb/viewer-data.json',{cache:'no-store'}).then(response=>{
  if(!response.ok)throw new Error('viewer data unavailable');return response.json()
}).then(data=>KBUI.start(data)).catch(error=>{
  const results=document.getElementById('results');results.hidden=false;results.textContent=error.message
});
})();
