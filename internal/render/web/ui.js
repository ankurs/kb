(function(global){
'use strict';
function pathURL(path){return path.split('/').map(encodeURIComponent).join('/')}
function excalidrawPreview(href){
  if(/^(?:[a-z][a-z0-9+.-]*:|\/\/)/i.test(href))return href;
  return href.replace(/\.excalidraw(?=[?#]|$)/i,extension=>extension+'.png')
}
function routePath(){
  let route=location.hash.replace(/^#\/?/,'').split('#',1)[0];
  try{return decodeURIComponent(route)}catch(e){return route}
}
function plainText(md){
  return String(md).replace(/```/g,' ').replace(/`/g,'')
    .replace(/!\[([^\]]*)\]\([^)]*\)/g,'$1')
    .replace(/\[([^\]]*)\]\([^)]*\)/g,'$1')
    .replace(/<[^>]*>/g,' ').replace(/^[ \t]*[#>*+|\-]+/gm,' ')
    .replace(/[*~]/g,'').replace(/\s+/g,' ').trim()
}
function labelsFrom(nav){
  let labels={};
  function walk(node){
    if(node.path)labels[node.path]=node.name;
    (node.children||[]).forEach(walk)
  }
  walk(nav||{});
  return labels
}
function entries(data){
  let labels=labelsFrom(data.nav);
  return Object.keys(data.pages||{}).sort().map((path,order)=>{
    let heading=/^#\s+(.+)$/m.exec(data.pages[path])?.[1]||'';
    let navLabel=labels[path]||'',label=(!navLabel||navLabel==='Home'||navLabel==='Overview')&&heading?plainText(heading):navLabel||path;
    let text=plainText(data.pages[path]);
    return {path,label,text,lowerLabel:(label+' '+heading).toLocaleLowerCase(),lowerText:text.toLocaleLowerCase(),order}
  })
}
function search(index,query){
  let terms=[...new Set(query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean))];
  if(query.trim().length<2)return [];
  return index.map(page=>{
    if(!terms.every(term=>page.lowerLabel.includes(term)||page.lowerText.includes(term)))return null;
    let score=0;
    terms.forEach(term=>{
      if(page.lowerLabel.includes(term))score+=5;
      let at=0,count=0;
      while((at=page.lowerText.indexOf(term,at))>=0&&count<50){count++;at+=term.length}
      score+=count
    });
    return {page,score,terms}
  }).filter(Boolean).sort((a,b)=>b.score-a.score||a.page.order-b.page.order).slice(0,40)
}
function highlight(parent,text,terms){
  let lower=text.toLocaleLowerCase(),at=0;
  while(at<text.length){
    let next=-1,match='';
    terms.forEach(term=>{let pos=lower.indexOf(term,at);if(pos>=0&&(next<0||pos<next||(pos===next&&term.length>match.length))){next=pos;match=term}});
    if(next<0){parent.appendChild(document.createTextNode(text.slice(at)));break}
    if(next>at)parent.appendChild(document.createTextNode(text.slice(at,next)));
    let mark=document.createElement('mark');mark.textContent=text.slice(next,next+match.length);parent.appendChild(mark);
    at=next+match.length
  }
}
function excerpt(page,terms){
  let at=-1;
  terms.forEach(term=>{let pos=page.lowerText.indexOf(term);if(pos>=0&&(at<0||pos<at))at=pos});
  if(at<0)return page.text.slice(0,140);
  let start=Math.max(0,at-50),end=Math.min(page.text.length,at+110);
  return (start?'…':'')+page.text.slice(start,end)+(end<page.text.length?'…':'')
}
function start(data){
  let nav=document.getElementById('nav'),results=document.getElementById('results');
  let input=document.getElementById('filter'),sidebar=document.getElementById('kb-sidebar');
  let menu=document.getElementById('menu-toggle'),index=entries(data);
  if(!nav||!results||!input||!sidebar||!menu)throw new Error('viewer shell incomplete');
  // Announce the result count without reading every matching snippet aloud.
  results.removeAttribute('aria-live');
  let count=document.createElement('p');count.className='result-count';count.setAttribute('role','status');
  results.appendChild(count);
  const mobile=global.matchMedia('(max-width:800px)');
  function syncMenu(){
    if(!mobile.matches)sidebar.classList.remove('open');
    menu.setAttribute('aria-expanded',String(mobile.matches&&sidebar.classList.contains('open')));
    // Offscreen links and search must not remain in the keyboard tab order.
    sidebar.inert=mobile.matches&&!sidebar.classList.contains('open')
  }
  function closeMenu(){
    if(mobile.matches&&sidebar.contains(document.activeElement))menu.focus();
    sidebar.classList.remove('open');syncMenu()
  }
  function openMenu(){sidebar.classList.add('open');syncMenu();input.focus()}
  syncMenu();
  mobile.addEventListener('change',()=>{
    if(mobile.matches&&sidebar.contains(document.activeElement))menu.focus();
    syncMenu()
  });
  function link(node){
    let li=document.createElement('li'),a=document.createElement('a');
    a.href='#/'+pathURL(node.path);a.dataset.path=node.path;a.textContent=node.name;
    a.addEventListener('click',()=>{closeMenu();setActive(node.path)});
    li.appendChild(a);return li
  }
  function branch(node){
    if(node.path)return link(node);
    let li=document.createElement('li'),details=document.createElement('details');
    let summary=document.createElement('summary');summary.textContent=node.name;
    let children=document.createElement('ul');(node.children||[]).forEach(child=>children.appendChild(branch(child)));
    details.append(summary,children);li.appendChild(details);return li
  }
  let list=document.createElement('ul');(data.nav?.children||[]).forEach(child=>list.appendChild(branch(child)));
  nav.replaceChildren(list);
  function setActive(path){
    let current=path||routePath()||((data.pages||{})['README.md']?'README.md':index[0]?.path);
    nav.querySelectorAll('a[data-path]').forEach(a=>{
      let active=a.dataset.path===current;a.classList.toggle('kb-active',active);
      if(active){
        a.setAttribute('aria-current','page');
        for(let p=a.closest('details');p;p=p.parentElement?.closest('details'))p.open=true;
        // Keep the selected page visible when navigation changes from another
        // section, including after a search result click.
        if(!nav.hidden&&getComputedStyle(sidebar).transform==='none')a.scrollIntoView({block:'nearest'})
      }
      else a.removeAttribute('aria-current')
    })
  }
  function showResults(){
    let query=input.value.trim();
    if(query.length<2){results.hidden=true;count.textContent='';results.replaceChildren(count);nav.hidden=false;setActive();return}
    let matches=search(index,query);
    count.textContent=matches.length===40?'Showing top 40 results':
      matches.length?`${matches.length} result${matches.length===1?'':'s'}`:'No matches';
    let list=document.createElement('div');
    matches.forEach(({page,terms})=>{
      let a=document.createElement('a');a.className='result';a.href='#/'+pathURL(page.path);a.dataset.path=page.path;
      let title=document.createElement('strong');title.className='result-title';highlight(title,page.label,terms);
      let snippet=document.createElement('span');snippet.className='result-snippet';highlight(snippet,excerpt(page,terms),terms);
      a.append(title,snippet);a.addEventListener('click',closeMenu);list.appendChild(a)
    });
    results.replaceChildren(count,list);results.hidden=false;nav.hidden=true
  }
  menu.addEventListener('click',()=>sidebar.classList.contains('open')?closeMenu():openMenu());
  input.addEventListener('input',showResults);
  input.addEventListener('keydown',e=>{
    if(e.key==='Escape'){
      e.preventDefault();e.stopPropagation();input.value='';showResults();
      if(mobile.matches&&sidebar.classList.contains('open'))closeMenu();
      else input.blur()
    }
  });
  document.addEventListener('keydown',e=>{
    if(e.key==='Escape'&&mobile.matches&&sidebar.classList.contains('open')){
      e.preventDefault();closeMenu();return
    }
    if((e.metaKey||e.ctrlKey)&&!e.altKey&&!e.shiftKey&&e.key.toLocaleLowerCase()==='k'){
      e.preventDefault();if(mobile.matches)openMenu();
      input.focus();input.select()
    }
  });
  window.addEventListener('hashchange',()=>setActive());setActive();
  let api={setActive,showResults,search:q=>search(index,q)};global.kbViewerUI=api;return api
}
function externalLinks(root,openInNewTab){
  if(!root)return;
  root.querySelectorAll('a[href]').forEach(a=>{
    let href=a.getAttribute('href')||'';
    if(!/^(?:https?:\/\/|\/\/)/i.test(href))return;
    if(!openInNewTab){a.removeAttribute('target');return}
    a.setAttribute('target','_blank');
    let rel=new Set((a.getAttribute('rel')||'').split(/\s+/).filter(Boolean));
    rel.add('noopener');rel.add('noreferrer');a.setAttribute('rel',[...rel].join(' '))
  })
}
global.KBUI={start,externalLinks,excalidrawPreview};
})(window);
