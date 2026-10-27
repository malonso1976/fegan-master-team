'use strict';
let data={clubs:[],swimmers:[],marks:[]},token='',events=[],homeClubID='',catalog=null,syncTimer=null;
const $=s=>document.querySelector(s), $$=s=>[...document.querySelectorAll(s)];
const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const fmt=n=>n>=6000?`${Math.floor(n/6000)}:${String(Math.floor(n/100)%60).padStart(2,'0')}.${String(n%100).padStart(2,'0')}`:`${Math.floor(n/100)}.${String(n%100).padStart(2,'0')}`;
const signed=n=>(n<0?'−':'+')+fmt(Math.abs(n));
const eventName=e=>e.replace(/([LMEBS])$/,s=>' '+({L:'libre',M:'mariposa',E:'espalda',B:'braza',S:'estilos'}[s]));
let markSort={key:'date',direction:-1};
const normalized=value=>String(value??'').normalize('NFD').replace(/[\u0300-\u036f]/g,'').toLocaleLowerCase('es');
const collator=new Intl.Collator('es',{sensitivity:'base',numeric:true});
const clubTitle=c=>`${c.medium_name||c.name} · ${c.id}`;
const sortedClubs=clubs=>[...clubs].sort((a,b)=>collator.compare(a.medium_name||a.name,b.medium_name||b.name)||collator.compare(a.id,b.id));
function clubName(id,extra=[]){const c=data.clubs.find(c=>c.id===id)||extra.find(c=>c.id===id)||catalog?.clubs?.find(c=>c.id===id);return c?clubTitle(c):id;}
const swimmerName=l=>data.swimmers.find(s=>s.license===l)?.name||l;
const chronoName=c=>c==='E'?'Electrónico':c==='M'?'Manual':'Manual y electrónico';
const genderName=g=>({M:'Masculino',F:'Femenino',X:'Mixto'}[g]||'Categoría desconocida');
const relayName=m=>`Relevo ${genderName(m.gender).toLocaleLowerCase('es')} · +${m.relay_category||'?'}${m.age_total?' · '+m.age_total+' años':''}`;
function notice(message,error=false){const n=$('#notice');n.textContent=message;n.hidden=false;n.classList.toggle('error',error);}
async function api(path,body){const multipart=body instanceof FormData;const r=await fetch('/api/'+path,{method:body===undefined?'GET':'POST',headers:body===undefined?{}:multipart?{'X-CSRF-Token':token}:{'Content-Type':'application/json','X-CSRF-Token':token},body:body===undefined?undefined:multipart?body:JSON.stringify(body)});const v=await r.json();if(!r.ok)throw Error(v.error||'No se pudo completar la operación');return v;}
function tab(id){if(!homeClubID&&id!=='setup'&&id!=='settings'){notice('Elige primero el club de tu equipo.');id='setup';}$$('main>section').forEach(s=>s.hidden=s.id!==id);$$('nav button').forEach(b=>b.dataset.tab===id?b.setAttribute('aria-current','page'):b.removeAttribute('aria-current'));window.scrollTo(0,0);}
document.addEventListener('click',e=>{const b=e.target.closest('[data-tab]');if(b)tab(b.dataset.tab);});
function options(select,items,blank=''){const old=select.value;select.innerHTML=(blank?`<option value="">${esc(blank)}</option>`:'')+items.map(([v,n])=>`<option value="${esc(v)}">${esc(n)}</option>`).join('');if([...select.options].some(o=>o.value===old))select.value=old;}
function table(headers,rows,empty='Todavía no hay datos.'){if(!rows.length)return `<p class="empty">${esc(empty)}</p>`;return `<table><thead><tr>${headers.map(h=>typeof h==='object'?`<th scope="col" aria-sort="${markSort.key===h.key?(markSort.direction===1?'ascending':'descending'):'none'}"><button type="button" data-mark-sort="${esc(h.key)}" aria-label="Ordenar por ${esc(h.label)}">${esc(h.label)} ${markSort.key===h.key?(markSort.direction===1?'↑':'↓'):'↕'}</button></th>`:`<th scope="col">${esc(h)}</th>`).join('')}</tr></thead><tbody>${rows.map(row=>`<tr>${row.map(v=>`<td>${v}</td>`).join('')}</tr>`).join('')}</tbody></table>`;}
function markRows(marks,edit=false){return marks.map(m=>[esc(m.date),esc(m.license?swimmerName(m.license):relayName(m)),esc(eventName(m.event)),`<strong>${fmt(m.centiseconds)}</strong>`,esc(m.pool+' m · '+chronoName(m.chrono)),esc(clubName(m.club_id)),`<span class="badge ${m.source==='manual'?'manual':''}">${esc(m.source)}</span> ${m.verified?'✓':'sin verificar'}`,...(edit?[`<button data-edit-mark="${esc(m.id)}">Editar</button> <button class="danger" data-delete-kind="marks" data-delete-id="${esc(m.id)}">Borrar</button>`]:[])]);}
function markValue(mark,key){
 switch(key){
 case 'swimmer':return mark.license?swimmerName(mark.license):relayName(mark);
 case 'event':return eventName(mark.event);
 case 'club':return clubName(mark.club_id);
 case 'origin':return mark.source+' '+(mark.verified?'verificada':'sin verificar');
 case 'pool':return mark.pool;
 case 'time':return mark.centiseconds;
 default:return mark.date;
 }
}
function renderMarks(){
 const search=normalized($('#mark-search').value),from=$('#mark-from').value,to=$('#mark-to').value,event=$('#mark-event').value,gender=$('#mark-gender').value,type=$('#mark-type').value;
 $('#mark-best').disabled=type==='relay';
 if(type==='relay')$('#mark-best').checked=false;
 const bestOnly=$('#mark-best').checked;
 let marks=data.marks.filter(m=>(!from||m.date>=from)&&(!to||m.date<=to)&&(!event||m.event===event)&&(!gender||m.gender===gender)&&(!type||(m.event.startsWith('4x')?'relay':'individual')===type)&&normalized([m.license,swimmerName(m.license),clubName(m.club_id),m.event,eventName(m.event),genderName(m.gender),m.event.startsWith('4x')?relayName(m):'individual'].join(' ')).includes(search));
 if(bestOnly){const best=new Map();for(const m of marks){if(!m.license||m.event.startsWith('4x'))continue;const key=JSON.stringify([m.license,m.event,m.pool,m.chrono,m.gender]),old=best.get(key);if(!old||m.centiseconds<old.centiseconds||(m.centiseconds===old.centiseconds&&(m.date>old.date||(m.date===old.date&&m.id<old.id))))best.set(key,m);}marks=[...best.values()];}
 marks.sort((a,b)=>{const x=markValue(a,markSort.key),y=markValue(b,markSort.key);let cmp=typeof x==='number'?x-y:collator.compare(x,y);if(!cmp&&markSort.key==='pool')cmp=collator.compare(a.chrono,b.chrono);return markSort.direction*cmp||collator.compare(b.date,a.date)||collator.compare(a.id,b.id);});
 const headers=[['date','Fecha'],['swimmer','Nadador / relevo'],['event','Prueba'],['time','Tiempo'],['pool','Piscina / crono'],['club','Club'],['origin','Origen']].map(([key,label])=>({key,label}));
 $('#mark-list').innerHTML=table([...headers,''],markRows(marks,true));
 $('#mark-summary').textContent=`${marks.length} ${marks.length===1?'marca':'marcas'}${bestOnly?' · mejores individuales dentro de los filtros':''}`;
}
function renderRecommendationSwimmers(){
 const words=normalized($('#recommend-search').value).trim().split(/\s+/).filter(Boolean);
 const people=data.swimmers.filter(p=>{const club=data.clubs.find(c=>c.id===p.club_id);const text=normalized([p.name,p.license,p.club_id,club?.name,club?.medium_name,club?.short_name].join(' '));return words.every(word=>text.includes(word));}).sort((a,b)=>collator.compare(a.name,b.name)||collator.compare(a.license,b.license));
 const select=$('#recommend-form').elements.license;
 options(select,people.map(p=>[p.license,`${p.name} · ${p.license} · ${clubName(p.club_id)}`]),'Selecciona un nadador');
 if(people.length===1&&words.length)select.value=people[0].license;
 $('#recommend-count').textContent=`${people.length} ${people.length===1?'nadador encontrado':'nadadores encontrados'}`;
}

const selectedMembers=new Set();let memberBulkBusy=false;
function visibleMembers(){const search=$('#member-search').value.toLocaleLowerCase();return data.swimmers.filter(s=>[s.name,s.license,clubName(s.club_id)].join(' ').toLocaleLowerCase().includes(search));}
function updateMemberSelection(){const visible=visibleMembers();for(const license of selectedMembers)if(!visible.some(s=>s.license===license))selectedMembers.delete(license);const all=$('#member-select-all');all.checked=visible.length>0&&selectedMembers.size===visible.length;all.indeterminate=selectedMembers.size>0&&!all.checked;all.disabled=memberBulkBusy||!visible.length;$('#member-selection-count').textContent=`${selectedMembers.size} seleccionados`;$('#member-deactivate').disabled=memberBulkBusy||!selectedMembers.size;$$('[data-select-member]').forEach(c=>{c.checked=selectedMembers.has(c.dataset.selectMember);c.disabled=memberBulkBusy;});}
function renderMembers(){const search=$('#member-search').value.toLocaleLowerCase();$('#member-list').innerHTML=table(['Seleccionar','Nombre','Licencia','Nacimiento','Categoría','Club','Estado',''],data.swimmers.filter(s=>[s.name,s.license,clubName(s.club_id)].join(' ').toLocaleLowerCase().includes(search)).sort((a,b)=>collator.compare(a.name,b.name)).map(s=>[`<input type="checkbox" data-select-member="${esc(s.license)}" aria-label="${esc('Seleccionar '+s.name+' · '+s.license)}" ${selectedMembers.has(s.license)?'checked':''}>`,esc(s.name),esc(s.license),s.birth_year,esc(s.gender),esc(clubName(s.club_id)),s.member?(s.available?'Equipo · disponible':'Equipo · no disponible'):'Competidor',`<button data-edit-swimmer="${esc(s.license)}">Editar</button> ${s.member?`<button data-import-history="${esc(s.license)}">Historial de 10 años</button> `:''}<button class="danger" data-delete-kind="swimmers" data-delete-id="${esc(s.license)}">Borrar</button>`]));updateMemberSelection();}
async function refresh(){
 data=await api('data');
 $$('select.clubs').forEach(s=>options(s,sortedClubs(data.clubs).map(c=>[c.id,clubTitle(c)]),'Selecciona un club'));
 const people=[...data.swimmers].sort((a,b)=>collator.compare(a.name,b.name));
 renderRecommendationSwimmers();
 options($('#history-member'),people.filter(p=>p.member).map(p=>[p.license,`${p.name} · ${p.license} · ${clubName(p.club_id)}`]),'Selecciona un miembro');
 $('#swimmer-options').innerHTML=people.map(p=>`<option value="${esc(p.license)}" label="${esc(p.name+' · '+p.license)}"></option>`).join('');
 $('#club-options').innerHTML=sortedClubs(data.clubs).map(c=>`<option value="${esc(c.id)}" label="${esc(clubTitle(c))}"></option>`).join('');
 $('#stats').innerHTML=[[data.swimmers.filter(s=>s.member).length,'Nadadores del equipo'],[data.marks.length,'Marcas guardadas'],[data.clubs.length,'Clubes']].map(([n,s])=>`<div class="stat"><strong>${n}</strong><span>${s}</span></div>`).join('');
 $('#recent').innerHTML=table(['Fecha','Nadador / relevo','Prueba','Tiempo','Piscina / crono','Club','Origen'],markRows(data.marks.slice(0,8)));renderMembers();renderMarks();
 $('#club-list').innerHTML=table(['Club','Identificador','Nombre medio','Nombre corto',''],sortedClubs(data.clubs).map(c=>[esc(c.name),esc(c.id),esc(c.medium_name||'—'),esc(c.short_name||'—'),`<button data-edit-club="${esc(c.id)}">Editar</button> <button class="danger" data-delete-kind="clubs" data-delete-id="${esc(c.id)}">Borrar</button>`]));
 $('#status').textContent=homeClubID?clubName(homeClubID)+' · datos locales':'Datos locales · elige tu club';
}
function values(form){const out=Object.fromEntries(new FormData(form));form.querySelectorAll('input[type=checkbox]').forEach(c=>out[c.name]=c.checked);for(const name of ['birth_year','pool','season_start','category','relay_category','limit','age_total','birth_from','birth_to'])if(name in out)out[name]=Number(out[name]);return out;}
function bind(id,fn){const form=$(id);form.addEventListener('submit',async e=>{e.preventDefault();const button=form.querySelector('button[type=submit],button.primary');button.disabled=true;try{await fn(values(form),form);}catch(e){notice(e.message,true);}finally{button.disabled=false;}});}
function fill(form,item){for(const [key,value]of Object.entries(item)){const el=form.elements.namedItem(key);if(!el)continue;if(el.type==='checkbox')el.checked=value;else el.value=Array.isArray(value)?value.join(', '):value??'';}}
function parseTime(s){const match=s.trim().match(/^(?:(\d{1,3}):)?(\d{1,3})(?:[.,](\d{1,2}))?$/);if(!match||(match[1]&&+match[2]>=60))throw Error('Tiempo esperado: 28.45 o 1:02.34');return ((+match[1]||0)*60+(+match[2]))*100+Number((match[3]||'').padEnd(2,'0'));}
function seasonOn(date){const [year,month]=date.split('-').map(Number);return month>=10?year:year-1;}
function selectedSwimmer(){const text=$('#mark-form').elements.license.value.trim();const matches=data.swimmers.filter(s=>s.license===text||s.name.toLocaleLowerCase()===text.toLocaleLowerCase());return matches.length===1?matches[0]:null;}
function updateMarkSwimmer(useCurrentClub=false,normalize=false){const f=$('#mark-form'),s=selectedSwimmer();if(s&&normalize)f.elements.license.value=s.license;if(useCurrentClub)f.elements.club_id.value=s?.club_id||'';const age=s?seasonOn(f.elements.date.value||today())+1-s.birth_year:0;$('#mark-category').value=s?`${s.gender==='F'?'Femenina':'Masculina'} · ${age>=20?(age>=100?'100+':`${Math.floor(age/5)*5}–${Math.floor(age/5)*5+4}`):'Menor de 20'} · ${s.name}`:'';}
$('#mark-form').elements.license.addEventListener('change',()=>updateMarkSwimmer(true,true));
$('#mark-form').elements.license.addEventListener('input',()=>updateMarkSwimmer(true));
$('#mark-form').elements.date.addEventListener('change',()=>updateMarkSwimmer());
bind('#club-form',async v=>{await api('clubs',v);await refresh();notice('Club guardado.');});
bind('#swimmer-form',async v=>{await api('swimmers',v);await refresh();notice('Nadador guardado.');if(homeClubID)pollSync();});
for(const id of ['#mark-form','#relay-mark-form']){
 bind(id,async(v,f)=>{v.centiseconds=parseTime(v.time);delete v.time;v.source='manual';if(id==='#mark-form'){const s=selectedSwimmer();if(!s)throw Error('Selecciona un nadador conocido por nombre o licencia.');v.license=s.license;v.gender=s.gender;}else{v.license='';v.relay_licenses=v.relay_licenses.split(',').map(x=>x.trim()).filter(Boolean);}await api('marks',v);f.reset();await refresh();notice('Marca guardada.');if(homeClubID)pollSync();});
 $(id).addEventListener('reset',()=>{setTimeout(()=>{$(id).elements.id.value='';$(id).elements.club_id.value='';if(id==='#mark-form')updateMarkSwimmer();},0);});
}
$('#swimmer-form').addEventListener('reset',()=>$('#swimmer-form').elements.license.readOnly=false);
$('#club-form').addEventListener('reset',()=>$('#club-form').elements.id.readOnly=false);
document.addEventListener('click',async e=>{const b=e.target.closest('button');if(!b)return;
 if(b.dataset.importHistory){prepareMemberHistory(b.dataset.importHistory);return;}
 if(b.dataset.editSwimmer){tab('members');fill($('#swimmer-form'),data.swimmers.find(s=>s.license===b.dataset.editSwimmer));$('#swimmer-form').elements.license.readOnly=true;$('#swimmer-form').scrollIntoView({block:'center'});}
 if(b.dataset.editClub){tab('clubs');fill($('#club-form'),data.clubs.find(c=>c.id===b.dataset.editClub));$('#club-form').elements.id.readOnly=true;$('#club-form').scrollIntoView({block:'center'});}
 if(b.dataset.editMark){const m=data.marks.find(m=>m.id===b.dataset.editMark),f=$(m.event.startsWith('4x')?'#relay-mark-form':'#mark-form');f.reset();setTimeout(()=>{fill(f,{...m,time:fmt(m.centiseconds),relay_licenses:m.relay_licenses||[]});f.closest('details').open=true;if(!m.event.startsWith('4x'))updateMarkSwimmer();f.scrollIntoView({block:'center'});},0);}
 if(b.dataset.deleteKind){const kind=b.dataset.deleteKind,id=b.dataset.deleteId;const people=new Set(data.swimmers.filter(s=>(kind==='swimmers'&&s.license===id)||(kind==='clubs'&&s.club_id===id)).map(s=>s.license));const marks=data.marks.filter(m=>(kind==='marks'&&m.id===id)||(kind==='clubs'&&m.club_id===id)||people.has(m.license)||(m.relay_licenses||[]).some(l=>people.has(l)));const label=kind==='clubs'?clubName(id):kind==='swimmers'?swimmerName(id):'esta marca';if(!confirm(`¿Borrar ${label}?\nSe eliminarán ${people.size} nadadores y ${marks.length} marcas relacionadas, con sus versiones anteriores. Esta acción no se puede deshacer. Una importación automática no los restaurará.`))return;b.disabled=true;try{const result=await api('delete',{kind,id});if(result.home_cleared)homeClubID='';await refresh();for(const f of ['#mark-form','#relay-mark-form','#swimmer-form','#club-form'])$(f).reset();$('#history').innerHTML='';for(const id of ['#recommend-results','#relay-results','#preview','#pdf-preview']){$(id).innerHTML='';$(id).hidden=true;}notice(`Borrado completado: ${result.clubs} clubes, ${result.swimmers} nadadores y ${result.marks} marcas.`);if(!homeClubID){renderCatalog();tab('setup');}}catch(e){notice(e.message,true);b.disabled=false;}}
});
$('#import-form').elements.club_id.addEventListener('input',()=>{$('#import-club-name').textContent=clubName($('#import-form').elements.club_id.value);});
$('#member-search').addEventListener('input',()=>{selectedMembers.clear();renderMembers();});
$('#member-select-all').addEventListener('change',e=>{for(const swimmer of visibleMembers()){if(e.target.checked)selectedMembers.add(swimmer.license);else selectedMembers.delete(swimmer.license);}updateMemberSelection();});
document.addEventListener('change',e=>{const license=e.target.dataset.selectMember;if(!license)return;if(e.target.checked)selectedMembers.add(license);else selectedMembers.delete(license);updateMemberSelection();});
$('#member-deactivate').addEventListener('click',async()=>{if(memberBulkBusy||!selectedMembers.size)return;memberBulkBusy=true;const licenses=[...selectedMembers];updateMemberSelection();try{const result=await api('swimmers/deactivate',{licenses});selectedMembers.clear();$('#swimmer-form').reset();for(const id of ['#relay-results','#recommend-results'])$(id).hidden=true;await refresh();notice(`${result.updated} nadadores actualizados: fuera del equipo y no disponibles para relevos. Sus fichas y marcas se conservan.`);}catch(e){notice(e.message,true);}finally{memberBulkBusy=false;updateMemberSelection();}});
$('#recommend-search').addEventListener('input',()=>{renderRecommendationSwimmers();$('#recommend-results').hidden=true;});
$('#recommend-form').elements.license.addEventListener('change',()=>$('#recommend-results').hidden=true);
document.addEventListener('click',event=>{const button=event.target.closest('[data-mark-sort]');if(!button)return;const key=button.dataset.markSort;markSort={key,direction:markSort.key===key?-markSort.direction:(key==='date'?-1:1)};renderMarks();$(`[data-mark-sort="${key}"]`).focus();});
['#mark-search','#mark-from','#mark-to','#mark-event','#mark-gender','#mark-type','#mark-best'].forEach(s=>$(s).addEventListener('input',renderMarks));
$('#history-load').addEventListener('click',async()=>{try{const h=await api('history');$('#history').innerHTML=table(['Fecha','Tipo','Identificador','Versión anterior'],h.map(x=>[esc(x.at),esc(x.entity),esc(x.id),`<details><summary>Ver datos</summary><pre>${esc(JSON.stringify(x.before,null,2))}</pre></details>`]));}catch(e){notice(e.message,true);}});
function showPreview(selector,result){const p=result.preview,box=$(selector);box.hidden=false;const warnings=[...new Set(p.warnings||[])];box.innerHTML=`<h2>Revisa la importación</h2><p>${p.data.swimmers.length} nadadores · ${p.data.marks.length} marcas${p.requests?' · '+p.requests+' consultas':''}</p><ul class="muted">${warnings.map(w=>`<li>${esc(w)}</li>`).join('')}</ul>${table(['Nadador / categoría','Club','Prueba','Tiempo','Fecha','Piscina / crono'],p.data.marks.map(m=>[esc(m.license?(p.data.swimmers.find(s=>s.license===m.license)?.name||m.license)+' · '+m.license:relayName(m)),esc(clubName(m.club_id,p.data.clubs)),esc(eventName(m.event)),fmt(m.centiseconds),esc(m.date),esc(m.pool+' m · '+chronoName(m.chrono))]),'No hay resultados importables. Revisa los avisos y las correspondencias de clubes.')}<p>Los duplicados y los registros borrados previamente se omiten. Las fichas existentes y las correcciones manuales se conservan.</p><button data-confirm-preview="${esc(result.id)}" class="primary" ${p.data.marks.length?'':'disabled'}>Guardar estos resultados</button>`;box.scrollIntoView({block:'start'});}
function prepareMemberHistory(license){
 const swimmer=data.swimmers.find(s=>s.license===license&&s.member);
 if(!swimmer)throw Error('Selecciona un miembro del club.');
 const form=$('#import-form'),date=new Date(),month=date.getMonth();date.setFullYear(date.getFullYear()-10);if(date.getMonth()!==month)date.setDate(0);
 const from=[date.getFullYear(),String(date.getMonth()+1).padStart(2,'0'),String(date.getDate()).padStart(2,'0')].join('-');
 form.reset();fill(form,{club_id:'',license:swimmer.license,gender:swimmer.gender,event:'all',from,to:today(),birth_from:'',birth_to:'',member:false});
 $('#import-season').value='';$('#import-season-label').textContent='Últimos diez años · todos los clubes de esta licencia';$('#import-club-name').textContent='Se incluyen clubes anteriores';$('#preview').hidden=true;tab('import');
 $('#history-member').value=swimmer.license;
 notice(`Historial de ${swimmer.name}: ${from} a ${today()}. Pulsa Consultar y preparar importación; revisarás los resultados antes de guardarlos.`);
 form.scrollIntoView({block:'start'});
}
$('#history-prepare').addEventListener('click',()=>{try{prepareMemberHistory($('#history-member').value);}catch(e){notice(e.message,true);}});
bind('#import-form',async v=>{
 $('#preview').hidden=true;notice('Consultando FEGAN en segundo plano para ambas piscinas y cronometrajes…');let result=await api('import/start',v);const id=result.id;
 while(!result.ready){await new Promise(resolve=>setTimeout(resolve,2000));result=await api('import/status/'+encodeURIComponent(id));}
 if(result.error)throw Error(result.error);showPreview('#preview',result);notice('Consulta preparada. Revisa los resultados antes de guardarlos.');
});
bind('#pdf-form',async(v,f)=>{const file=f.elements.file.files[0];if(!file||file.size>20*1024*1024)throw Error('Selecciona un PDF de hasta 20 MiB.');const body=new FormData(f),aliases={};for(const line of v.aliases.split('\n').map(s=>s.trim()).filter(Boolean)){const at=line.lastIndexOf('=');if(at<1)throw Error('Alias esperado: nombre del PDF = código de club');aliases[line.slice(0,at).trim()]=line.slice(at+1).trim();}body.set('aliases',JSON.stringify(aliases));body.set('verified',String(v.verified));$('#pdf-preview').hidden=true;notice('Leyendo el PDF local…');showPreview('#pdf-preview',await api('relay-pdf/preview',body));notice('PDF leído. Revisa las marcas y avisos antes de guardar.');});
document.addEventListener('click',async e=>{const b=e.target.closest('[data-confirm-preview]');if(!b)return;b.disabled=true;try{const r=await api('import/confirm',{id:b.dataset.confirmPreview});await refresh();b.closest('.panel').hidden=true;notice(`Importación completada: ${r.inserted} nuevas marcas y ${r.duplicates} duplicados o registros borrados omitidos.`);}catch(e){notice(e.message,true);b.disabled=false;}});
function rivalDetails(r){const own={name:swimmerName(r.own_mark.license),license:r.own_mark.license,birth:data.swimmers.find(s=>s.license===r.own_mark.license)?.birth_year,mark:r.own_mark,own:true};const rows=[own,...r.details.map(d=>({name:d.swimmer.name,license:d.swimmer.license,birth:d.swimmer.birth_year,mark:d.mark}))].sort((a,b)=>a.mark.centiseconds-b.mark.centiseconds||collator.compare(a.license,b.license));const max=Math.max(...rows.map(p=>p.mark.centiseconds));return `<details><summary>Reconocer a los rivales y comparar marcas · ${r.rivals} ${r.rivals===1?'referencia':'referencias'}</summary><p class="muted">Diferencia = marca del rival − marca propia. Una diferencia negativa indica un rival más rápido.</p>${table(['Nadador','Licencia','Nacimiento','Club del resultado','Marca','Diferencia contigo','Fecha / lugar','Comparación visual'],rows.map(p=>[`${p.own?'<strong>Tu nadador · ':''}${esc(p.name)}${p.own?'</strong>':''}`,esc(p.license),esc(p.birth||'—'),esc(clubName(p.mark.club_id)),fmt(p.mark.centiseconds),p.own?'—':signed(p.mark.centiseconds-r.time),esc(p.mark.date+' · '+p.mark.location),`<progress class="time-bar ${p.own?'own':''}" max="${max}" value="${p.mark.centiseconds}" aria-label="${esc(p.name+' '+fmt(p.mark.centiseconds))}"></progress>`]))}</details>`;}
bind('#recommend-form',async v=>{$('#recommend-results').hidden=true;const rows=await api('recommend',v),box=$('#recommend-results');box.hidden=false;box.innerHTML='<h2>Oportunidades en el conjunto observado</h2><p class="muted">Mejor marca histórica guardada frente a hasta diez rivales distintos en los últimos dos años. Cada prueba usa la piscina elegida y el cronometraje de la mejor marca propia.</p>'+table(['Prueba','Piscina / crono','Mejor marca histórica','Fecha propia','Grupo','Puesto observado','Rivales','Diferencia al primero'],rows.map(r=>[esc(eventName(r.event)),`${r.pool} m · ${chronoName(r.chrono)}`,`<strong>${fmt(r.time)}</strong>`,esc(r.own_mark.date),esc(r.age_group),r.rivals?'#'+r.rank:'Sin comparación',r.rivals,signed(r.gap_to_first)]),'No hay marcas compatibles para este nadador y este periodo.')+rows.map(r=>`<article class="comparison"><h3>${esc(eventName(r.event))} · ${r.pool} m · ${chronoName(r.chrono)}</h3>${rivalDetails(r)}</article>`).join('');});
function relayCard(r,i,category,event){return `<article class="panel relay-card"><div class="relay-top"><div><h2>Propuesta ${i+1} · +${category} · ${esc(eventName(event))}</h2><span>${r.age_total} años de suma real</span></div><div><strong>${fmt(r.total)}</strong><small>Tiempo estimado</small></div></div>${table(['Orden','Nadador','Licencia','Parcial de referencia','Cronometraje','Fecha de referencia','Edad'],r.legs.map((l,j)=>[`${j+1} · ${esc(eventName(l.event))}`,esc(l.name),esc(l.license),fmt(l.time),esc(chronoName(l.chrono)),esc(l.date),l.age]))}<p class="muted">${r.rivals?`Puesto observado ${r.rank} frente a ${r.rivals} clubes rivales en +${category}.`:'Sin marcas de relevos rivales comparables.'} ${r.gap_to_podium!=null?`Diferencia al tercer rival: ${signed(r.gap_to_podium)}.`:'Sin referencia suficiente del tercer rival.'}</p></article>`;}
function relayRivalDetails(group,request){
 const rivals=group.rivals||[],own=group.results[0];
 const coverage=(group.coverage||[]).map(c=>`${eventName(c.event)}: ${c.swimmers} ${c.swimmers===1?'nadador':'nadadores'} con marca compatible`).join(' · ');
 const rows=rivals.map(r=>({name:clubName(r.mark.club_id,[r.club]),mark:r.mark,time:r.mark.centiseconds}));
 if(own)rows.push({name:'Tu equipo · estimación',time:own.total,own:true});
 rows.sort((a,b)=>a.time-b.time||collator.compare(a.name,b.name));
 const max=Math.max(1,...rows.map(r=>r.time));
 return `<article class="comparison"><h3>+${group.category} · ${esc(genderName(request.gender))} · ${esc(eventName(request.event))}</h3><p class="muted">Referencias propias: ${esc(coverage)}. Se necesitan cuatro licencias distintas, edades elegibles y, en mixtos, dos mujeres y dos hombres.</p>${own?'':'<p>Sin composición propia con estos filtros. Puedes consultar los rivales, pero no calcular un puesto ni una diferencia con tu equipo. Revisa las referencias de cada integrante, fechas, disponibilidad, piscina y cronometraje.</p>'}<details><summary>Ver rivales comparables · ${rivals.length} ${rivals.length===1?'club':'clubes'}</summary><p class="muted">Mejor marca por club rival en la categoría inscrita y los filtros elegidos. Diferencia = tiempo rival − estimación propia. Un valor negativo indica un rival más rápido. La estimación propia suma marcas individuales.</p>${table(['Club','Marca / estimación','Cronometraje','Diferencia con tu equipo','Fecha / competición','Origen','Comparación visual'],rows.map(r=>[r.own?`<strong>${esc(r.name)}</strong>`:esc(r.name),fmt(r.time),r.own?'Según parciales':esc(chronoName(r.mark.chrono)),r.own||!own?'—':signed(r.time-own.total),r.own?'Estimación':esc(r.mark.date+' · '+r.mark.location),r.own?'Suma de cuatro marcas':esc(r.mark.source)+(r.mark.verified?' · verificada':' · sin verificar'),`<progress class="time-bar ${r.own?'own':''}" max="${max}" value="${r.time}" aria-label="${esc(r.name+' '+fmt(r.time))}"></progress>`]),'Sin marcas de relevos rivales compatibles con estos filtros. Se conserva la piscina; se admiten cronometrajes manuales y electrónicos sin ajustar los tiempos.')}</details></article>`;
}
bind('#relay-form',async v=>{const box=$('#relay-results');box.hidden=true;const comparisons=await api('relays',v);box.hidden=false;const eligible=comparisons.map(c=>({...c,advice:{...c.advice,groups:c.advice.groups.filter(g=>g.age_eligible)}})).filter(c=>c.advice.groups.length),visible=eligible.filter(c=>c.advice.groups.some(g=>g.results.length||(g.rivals||[]).length));box.innerHTML=`<h2>${esc(clubName(v.club_id))}</h2>`+(visible.length?'':eligible.length?'<p>Hay integrantes elegibles por edades, pero no hay composiciones ni referencias rivales con estos filtros. Revisa fechas, pruebas, verificación y marcas guardadas.</p>':'<p>No hay categorías con cuatro integrantes disponibles y edades/composición elegibles. Revisa el equipo, su disponibilidad, la temporada y la categoría seleccionada.</p>')+visible.map(({pool,chrono,advice})=>`<div class="panel"><h2>${pool} m · ${chronoName(chrono)}</h2><h3>${advice.recommended_category?'Categoría recomendada: +'+advice.recommended_category:'Comparación de categorías'}</h3><p>Temporada ${esc(advice.season)} · edades a ${esc(advice.age_at)}</p><p>${advice.warnings.map(esc).join(' ')}</p>${table(['Categoría','Mejor estimación','Puesto','Clubes rivales','Diferencia al tercer rival'],advice.groups.map(g=>{const r=g.results[0];return ['+'+g.category+(g.recommended?' · recomendada':''),r?fmt(r.total):'Sin composición',r&&r.rivals?'#'+r.rank:'Sin comparación',(g.rivals||[]).length,r&&r.gap_to_podium!=null?signed(r.gap_to_podium):'Datos insuficientes'];}))}${advice.groups.map(g=>relayRivalDetails(g,v)).join('')}</div>`+advice.groups.map(g=>g.results.map((r,i)=>relayCard(r,i,g.category,v.event)).join('')).join('')).join('');});

function seasonLabel(start){return `${start}/${String((start+1)%100).padStart(2,'0')}`;}
function updateSeasonLabel(input){input.parentElement.querySelector('.season-label').textContent=`${seasonLabel(+input.value)} · edad a 31/12/${+input.value+1}`;}
$$('.season-start').forEach(input=>input.addEventListener('input',()=>updateSeasonLabel(input)));
$('#import-season').addEventListener('change',e=>{const start=+e.target.value;$('#import-form').elements.from.value=`${start}-10-01`;$('#import-form').elements.to.value=`${start+1}-09-30`;$('#import-season-label').textContent=seasonLabel(start);});
function renderCatalog(){const search=$('#catalog-search').value.toLocaleLowerCase(),entries=new Map((catalog?.clubs||[]).map(c=>[c.id,c]));for(const c of data.clubs)entries.set(c.id,c);options($('#setup-club'),sortedClubs([...entries.values()]).filter(c=>[c.id,c.name,c.medium_name,c.short_name].join(' ').toLocaleLowerCase().includes(search)).map(c=>[c.id,clubTitle(c)+(c.short_name?' · '+c.short_name:'')]));if(homeClubID&&[...$('#setup-club').options].some(o=>o.value===homeClubID))$('#setup-club').value=homeClubID;showChosenClub();}
function showChosenClub(){const id=$('#setup-club').value,c=data.clubs.find(c=>c.id===id)||(catalog?.clubs||[]).find(c=>c.id===id);$('#chosen-club').textContent=c?`${c.name} · ${c.id} · medio: ${c.medium_name||'—'} · corto: ${c.short_name||'—'}`:'';}
async function loadCatalog(){try{catalog=await api('catalog',{});$('#catalog-source').textContent=`Catálogo oficial ${seasonLabel(catalog.season_start)} · ${catalog.clubs.length} clubes`;}catch(e){$('#catalog-source').textContent='No se pudo actualizar el catálogo: '+e.message+'. Puedes seleccionar un club guardado o crearlo manualmente.';}renderCatalog();}
function showSync(status){const banner=$('#sync-banner');banner.hidden=!status.started_at;if(banner.hidden)return;$('#sync-stage').textContent=status.stage;$('#sync-progress').max=status.total||1;$('#sync-progress').value=status.completed;$('#sync-count').textContent=`${status.completed}/${status.total} · ${status.inserted} marcas nuevas · ${status.duplicates} duplicados o borrados omitidos · ${status.reused||0} consultas reutilizadas${status.running?' · puedes seguir trabajando':''}`;$('#sync-warnings').innerHTML=(status.warnings||[]).map(x=>`<li>${esc(x)}</li>`).join('');banner.querySelector('details').hidden=!status.warnings?.length;}
async function pollSync(){if(syncTimer)clearTimeout(syncTimer);try{const status=await api('sync');showSync(status);await refresh();if(status.running)syncTimer=setTimeout(pollSync,4000);}catch(e){notice(e.message,true);}}
async function startSync(scope,force=true){const status=await api('sync',{scope,force});showSync(status);pollSync();}
bind('#setup-form',async v=>{await api('setup',v);homeClubID=v.club_id;await refresh();tab('overview');pollSync();});
bind('#setup-manual',async v=>{await api('clubs',v);await api('setup',{club_id:v.id});homeClubID=v.id;await refresh();tab('overview');pollSync();});
$('#setup-club').addEventListener('change',showChosenClub);$('#catalog-search').addEventListener('input',renderCatalog);$('#catalog-reload').addEventListener('click',loadCatalog);
document.addEventListener('click',async e=>{const button=e.target.closest('[data-sync]');if(!button)return;try{await startSync(button.dataset.sync);}catch(e){notice(e.message,true);}});
let resetting=false;
$('#reset-open').addEventListener('click',()=>{
 $('#reset-form').reset();$('#reset-confirm').disabled=true;
 const counts=[[data.clubs.length,'club','clubes'],[data.swimmers.length,'nadador','nadadores'],[data.marks.length,'marca','marcas']].map(([n,singular,plural])=>`${n} ${n===1?singular:plural}`);
 $('#reset-summary').textContent=`Se eliminarán ${counts[0]}, ${counts[1]} y ${counts[2]}, además del historial y los ajustes.`;
 $('#reset-dialog').showModal();$('#reset-form').elements.confirmation.focus();
});
$('#reset-cancel').addEventListener('click',()=>$('#reset-dialog').close());
$('#reset-dialog').addEventListener('cancel',event=>{if(resetting)event.preventDefault();});
$('#reset-form').elements.confirmation.addEventListener('input',event=>{$('#reset-confirm').disabled=event.target.value!=='REINICIAR';});
bind('#reset-form',async v=>{
 if(v.confirmation!=='REINICIAR')throw Error('Escribe REINICIAR para confirmar.');
 resetting=true;$('#reset-cancel').disabled=true;$('#reset-form').elements.confirmation.readOnly=true;
 try{await api('reset',v);window.location.reload();}
 catch(error){resetting=false;$('#reset-cancel').disabled=false;$('#reset-form').elements.confirmation.readOnly=false;$('#reset-dialog').close();throw error;}
});
// A live response tracks the page without relying on throttled background timers.
async function watchBrowser(){
 let leaving=false,controller,generation=0;
 const disconnect=()=>{leaving=true;generation++;controller?.abort();};
 window.addEventListener('pagehide',disconnect);
 window.addEventListener('pageshow',event=>{if(event.persisted){leaving=false;connect();}});
 async function connect(){
  const current=++generation;
  while(!leaving&&generation===current){
   controller=new AbortController();
   try{
    const response=await fetch('/api/browser/presence',{method:'POST',headers:{'X-CSRF-Token':token},signal:controller.signal});
    if(!response.ok)return;
    const reader=response.body.getReader();
    while(!(await reader.read()).done){}
   }catch(error){if(leaving||generation!==current)return;}
   if(!leaving&&generation===current)await new Promise(resolve=>setTimeout(resolve,500));
  }
 }
 await connect();
}
function today(){const d=new Date();return [d.getFullYear(),String(d.getMonth()+1).padStart(2,'0'),String(d.getDate()).padStart(2,'0')].join('-');}
(async()=>{try{
 const session=await api('session');token=session.token;events=session.events;if(session.auto_close)watchBrowser();
 options($('#mark-event'),[['','Todas las pruebas'],...events.map(e=>[e,eventName(e)])]);
 $$('select.record-individual-events').forEach(s=>options(s,events.filter(e=>!e.startsWith('4x')).map(e=>[e,eventName(e)])));
 $$('select.individual-events').forEach(s=>options(s,[['all','Todas las pruebas individuales'],...events.filter(e=>!e.startsWith('4x')).map(e=>[e,eventName(e)])]));
 $$('select.record-relay-events').forEach(s=>options(s,events.filter(e=>e.startsWith('4x')).map(e=>[e,eventName(e)])));
 $$('select.relay-events').forEach(s=>options(s,['4x50L','4x50S'].map(e=>[e,eventName(e)])));
 const categories=[80,100,120,160,200,240,280,320];$$('select.advice-categories').forEach(s=>options(s,[['0','Todas · recomendar categoría'],...categories.map(c=>[c,'+'+c])]));$$('select.record-categories').forEach(s=>options(s,categories.map(c=>[c,'+'+c]),'Selecciona categoría'));
 const now=new Date(),seasonStart=seasonOn(today());$$('.season-start').forEach(i=>{i.value=seasonStart;updateSeasonLabel(i);});$('#import-season').value=seasonStart;$('#import-season-label').textContent=seasonLabel(seasonStart);const lookback=new Date(now);lookback.setFullYear(now.getFullYear()-2);const from=[lookback.getFullYear(),String(lookback.getMonth()+1).padStart(2,'0'),String(lookback.getDate()).padStart(2,'0')].join('-');$$('input[name=from]').forEach(i=>i.value=from);$$('input[name=to],input[name=date]').forEach(i=>{i.value=today();i.defaultValue=today();});
 const setup=await api('setup');homeClubID=setup.home_club_id;catalog=setup.catalog;await refresh();renderCatalog();if(!homeClubID){tab('setup');await loadCatalog();}else{showSync(setup.sync);startSync('all',false).catch(e=>notice(e.message,true));}
}catch(e){notice(e.message,true);$('#status').textContent='No se pudo cargar';}})();
