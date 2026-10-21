import { writeFileSync, mkdirSync } from 'node:fs';
mkdirSync('design/previews', { recursive: true });
const target={document_id:process.env.OPENPENCIL_DOCUMENT_ID||'tab-1',page_id:process.env.OPENPENCIL_PAGE_ID||'0:3'};
const colors={
  'Light/Surface':'#FFFFFF','Light/Canvas':'#F4F5F7','Light/Text':'#20242C',
  'Light/Secondary':'#596272','Light/Muted':'#626D7E','Light/Accent':'#315ECA',
  'Light/Rule':'#E7EAF0','Light/Hover':'#ECEEF2','Light/Pressed':'#E1E5EB',
  'Light/Sidebar':'#F4F5F7','Light/Selected':'#DCE7FC','Light/SelectedHover':'#CFDDF8',
  'Light/SelectedText':'#234A9C','Light/SelectedSecondary':'#465873','Light/Indicator':'#315ECA',
  'Light/AccentHover':'#EAF0FC','Light/AccentPressed':'#DBE6FA',
  'Light/Focus':'#315ECA','Light/Border':'#8791A0',
  'Light/Success':'#237A57','Light/SuccessSubtle':'#E7F3EC',
  'Light/Danger':'#AA3542','Light/DangerHover':'#FBEDEF','Light/DangerPressed':'#F4D9DE',
  'Light/Disabled':'#626D7E',
  'Dark/Surface':'#20232A','Dark/Canvas':'#171A20','Dark/Text':'#EBEEF5',
  'Dark/Secondary':'#B3BDCD','Dark/Muted':'#9CA7B8','Dark/Accent':'#9ABBFF',
  'Dark/Rule':'#383E49','Dark/Hover':'#30353E','Dark/Pressed':'#3A414C',
  'Dark/Sidebar':'#171A20','Dark/Selected':'#334B77','Dark/SelectedHover':'#3E5888',
  'Dark/SelectedText':'#E0EAFF','Dark/SelectedSecondary':'#C3D1EA','Dark/Indicator':'#9ABBFF',
  'Dark/AccentHover':'#2A3852','Dark/AccentPressed':'#35486B',
  'Dark/Focus':'#9ABBFF','Dark/Border':'#919FB5',
  'Dark/Success':'#8ED5B3','Dark/SuccessSubtle':'#213B31',
  'Dark/Danger':'#FFADB8','Dark/DangerHover':'#482A31','Dark/DangerPressed':'#59333D',
  'Dark/Disabled':'#9CA7B8',
  'Status/Success':'#237A57','Status/Error':'#B43D45',
};
const scalars={'Space/4':4,'Space/8':8,'Space/12':12,'Space/16':16,'Space/24':24,'Space/32':32,
  'Type/Caption':12,'Type/Label':13,'Type/Body':14,'Type/Reading':16,'Type/Title':24,'Type/Word':28,
  'Radius/Window':12,'Radius/Control':8,'Radius/Nav':8,'Radius/Row':8,'Radius/Check':4,'Radius/Indicator':2,
  'Layout/PopupInset':16,'Layout/GhostInset':8,'Layout/SidebarInset':12};
const jobs=Object.entries({...colors,...scalars}).map(([name,value])=>({
 name:'create_variable',args:{...target,collection_id:'1858066909:5',name,type:typeof value==='number'?'FLOAT':'COLOR',value:String(value)}
}));
writeFileSync('artifacts/openpencil/tokens.json',JSON.stringify(jobs));
writeFileSync('design/tokens.json',JSON.stringify({colors,scalars},null,2));
const token=name=>`{designVar(${JSON.stringify(name)})}`;
const C=(role,dark=false)=>token(`${dark?'Dark':'Light'}/${role}`);
const S=n=>token(`Space/${n}`);
const escape=s=>JSON.stringify(s);
function text(s,size=14,role='Text',weight=400,extra='',dark=false){
 const type={12:'Caption',13:'Label',14:'Body',16:'Reading',24:'Title',28:'Word'}[size];
 const chinese=/[\u3400-\u9fff]/.test(s);
 const font=chinese?'Microsoft YaHei UI':'Segoe UI Variable Text';
 if(chinese) weight=weight>=600?700:400;
 return `<Text name=${escape(s.slice(0,36))} font="${font}" size=${type?token('Type/'+type):`{${size}}`} weight={${weight}} lineHeight={${Math.round(size*(size>=16?1.6:1.5))}} color=${C(role,dark)} ${extra}>{${escape(s)}}</Text>`;
}
function frame(content,props=''){return `<Frame ${props}>${content}</Frame>`;}
function row(content,props=''){return frame(content,`flex="row" items="center" gap=${S(8)} ${props}`);}
function col(content,props=''){return frame(content,`flex="col" gap=${S(8)} ${props}`);}
const spacer=()=>frame('','w="fill" h={1}');
const rule=(dark=false)=>frame('',`w="fill" h={1} bg=${C('Rule',dark)}`);
const paths={
 refresh:'M20 7v5h-5 M19 11a7 7 0 1 0-2 7 M4 12a7 7 0 0 1 12-5l4 5',
 close:'M6 6l12 12 M18 6L6 18',
 minus:'M5 12h14',maximize:'M6 6h12v12H6z',
 copy:'M9 8h11v13H9z M15 8V3H4v13h5',
 pin:'M9 3h6 M10 3v6l-4 4v2h12v-2l-4-4V3 M12 15v7',
 book:'M4 4h6a3 3 0 0 1 2 1 3 3 0 0 1 2-1h6v15h-6a3 3 0 0 0-2 1 3 3 0 0 0-2-1H4z M12 5v15',
 translate:'M3 5h12 M9 3v2 M5 8c1 4 4 7 8 9 M13 5c-1 6-4 10-10 13 M14 20l4-10 4 10 M16 16h4',
 settings:'M9 3h6l1 3 3 1 2 5-2 5-3 1-1 3H9l-1-3-3-1-2-5 2-5 3-1z M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0',
 search:'M16 10a6 6 0 1 1-12 0 6 6 0 0 1 12 0 M15 15l6 6',
 arrow:'M4 12h15 M14 7l5 5-5 5',
 swap:'M3 8h17 M16 4l4 4-4 4 M21 16H4 M8 12l-4 4 4 4',
 check:'M5 12l4 4L19 6',
 chevron:'M9 5l7 7-7 7',
 download:'M12 3v12 M7 10l5 5 5-5 M4 16v5h16v-5',
 spark:'M12 3l2.5 6.5L21 12l-6.5 2.5L12 21l-2.5-6.5L3 12l6.5-2.5z',
 more:'M5 12h.1 M12 12h.1 M19 12h.1',
 stop:'M7 7h10v10H7z',
 loader:'M12 4a8 8 0 1 1-8 8',
 grip:'M9 6h.1 M15 6h.1 M9 12h.1 M15 12h.1 M9 18h.1 M15 18h.1',
};
function icon(name,role='Secondary',dark=false,size=18){
 // SVG paints use the same token values; other paint-bearing nodes bind variables directly.
 const color=colors[`${dark?'Dark':'Light'}/${role}`];
 return `<svg name="${name}" viewBox="0 0 24 24" size={${size}}><path d="${paths[name]}" stroke="${color}" strokeWidth={1.65} strokeLinecap="round" strokeLinejoin="round" fill="none" /></svg>`;
}
function button(label,ico='',dark=false,active=false,tone){
 const role=tone||(active?'SelectedText':(['翻译','保存','重试'].includes(label)?'Accent':label==='移除'?'Danger':'Secondary'));
 return row((ico?icon(ico,role,dark):'')+(label?text(label,13,role,500,'',dark):''),`name="${label||ico} / ghost / ${role}" h={32} px=${token('Layout/GhostInset')} rounded=${token('Radius/Control')} ${active?`bg=${C('Selected',dark)}`:''}`);
}
function checkbox(label,on=true,dark=false){
 return row(frame(on?icon('check','Surface',dark,12):'',`w={14} h={14} rounded=${token('Radius/Check')} flex="row" items="center" justify="center" ${on?`bg=${C('Accent',dark)}`:`stroke=${C('Border',dark)} strokeWidth={1}`}`)+text(label,13,'Secondary',400,'',dark),'gap={8} h={32}');
}
function header(dark=false,pinned=false){
 return row(text('翻译',14,'Text',600,'',dark)+button('','refresh',dark)+spacer()+button('简体中文','',dark)+button('','pin',dark,pinned)+button('','close',dark),`name="Drag region · content aligned at 16" w="fill" h={48} pl=${token('Layout/PopupInset')} pr=${token('Layout/GhostInset')}`);
}
function popup(kind='word',dark=false){
 let body='',footer='',w=440,h=224;
 if(kind==='word'){
  body=col(text('bank',28,'Text',600,'',dark)+row(text('名词',12,'Secondary',400,'',dark)+text('当前语境',12,'Secondary',400,'',dark),'gap={12}')+text('银行；金融机构',18,'Text',400,'',dark),'gap={4} w="fill" items="start"');
  footer=button('收藏','book',dark)+spacer()+button('复制','copy',dark);
 }else if(kind==='sentence'){
  w=536;h=208;body=text('真正的发现之旅，不在于寻找新的风景，\n而在于拥有新的眼光。',18,'Text',400,'w="fill"',dark);
  footer=spacer()+button('复制','copy',dark);
 }else if(kind==='loading'){
  w=440;h=168;body=col(row(icon('loader','Accent',dark)+text('正在翻译',14,'Text',500,'',dark))+text('结合上下文，确认 bank 在此处的含义…',13,'Secondary',400,'w="fill"',dark),'gap={8}');
  footer=spacer()+button('停止','stop',dark);
 }else if(kind==='error'){
  w=440;h=184;body=col(text('暂时无法连接',16,'Text',600,'w="fill"',dark)+text('请求超时，请检查网络或模型配置。',13,'Secondary',400,'w="fill"',dark),'w="fill" items="start"');
  footer=button('模型设置','settings',dark)+spacer()+button('重试','refresh',dark);
 }else if(kind==='long'){
  w=536;h=360;body=col(text('这些模板可作为富有启发性的载体，帮助我们捕捉、收集和整理与项目相关、不断演变的见解与文字。',18,'Text',400,'w="fill"',dark)+
  text('设计不止关乎外观，也关乎信息如何被理解。清晰的层级使读者知道先看哪里；稳定的间距让内容之间的关系变得自然。',18,'Text',400,'w="fill"',dark)+
  text('当内容较长时，正文独立滚动，操作始终留在触手可及的位置。',18,'Text',400,'w="fill"',dark)+
  text('设计系统把这些规则连接起来，使每个页面拥有一致的阅读节奏。',18,'Text',400,'w="fill"',dark),'gap={16} w="fill"');
  footer=spacer()+button('复制','copy',dark);
 }
 return col(header(dark,kind==='long')+frame(row(body+(kind==='long'?frame('',`w={3} h={80} rounded={2} bg=${C('Rule',dark)}`):''),`w="fill" items="start" gap={12} ${kind==='long'?'h="fill" overflow="hidden"':''}`),`name="Body / shared 16px inset" w="fill" h="fill" px=${token('Layout/PopupInset')} pt=${S(8)} pb=${S(12)}`)+row(footer,`name="Footer / 8 inset + 8 hit padding" w="fill" h={44} px=${token('Layout/GhostInset')}`),
 `name="Popup / ${kind} / ${dark?'Dark':'Light'}" w={${w}} h={${h}} gap={0} bg=${C('Surface',dark)} rounded=${token('Radius/Window')} shadow="0 12 40 #1B243318, 0 2 6 #1B24330A"`);
}
function toolbar(dark=false){
 return row(icon('grip','Muted',dark,16)+button('翻译','translate',dark)+frame('',`w={1} h={16} bg=${C('Rule',dark)}`)+button('复制','copy',dark),`name="Selection toolbar" h={40} px={6} gap={2} rounded=${token('Radius/Window')} bg=${C('Surface',dark)} shadow="0 6 24 #1B243318"`);
}
function caption(kicker,title,subtitle){
 return col(text(kicker,12,'Accent',600)+text(title,24,'Text',600)+text(subtitle,13,'Secondary'),'gap={4} w="fill"');
}
function board(name,content,w=1144){
 return col(content,`name="${name}" w={${w}} h="hug" p={40} gap={32} bg=${C('Canvas')}`);
}
function labelled(label,body){return col(text(label,12,'Secondary',500)+body,'gap={12}');}
function navItem(label,ico,selected=false,state='Default',dark=false){
 const bg=selected?(state==='Hover'?'SelectedHover':'Selected'):state==='Hover'?'Hover':state==='Pressed'?'Pressed':null;
 return row(frame('',`name="Selection indicator" w={3} h={20} rounded=${token('Radius/Indicator')} ${selected?`bg=${C('Indicator',dark)}`:''}`)+icon(ico,selected?'SelectedText':'Secondary',dark)+text(label,14,selected?'SelectedText':'Secondary',selected?600:400,'',dark),
 `name="${label} / ${selected?'Selected':state}" w="fill" h={40} pl={5} pr={12} gap={8} rounded=${token('Radius/Nav')} ${bg?`bg=${C(bg,dark)}`:''} ${state==='Focus'?`stroke=${C('Focus',dark)} strokeWidth={2}`:''}`);
}
function shell(active,content,dark=false){
 const nav=(label,ico)=>navItem(label,ico,label===active,'Default',dark);
 const sidebar=col(col(nav('翻译','translate')+nav('词汇表','book')+nav('设置','settings'),'gap={4} w="fill"')+
 frame('','w={1} h="fill"')+
 col(text('划词翻译已开启',12,'Secondary',400,'',dark)+text('Ctrl + Alt + T',12,'Muted',400,'',dark),'gap={4} px={12}'),
 `w={164} h="fill" px=${token('Layout/SidebarInset')} pt=${S(16)} pb=${S(24)} bg=${C('Sidebar',dark)}`);
 return col(row(icon('spark','Accent',dark,18)+text('Squirrel',14,'Text',600,'',dark)+spacer()+
 button('','minus',dark)+button('','maximize',dark)+button('','close',dark),
 `name="Custom title bar / drag region" w="fill" h={44} pl={20} pr={8} gap={4}`)+
 row(sidebar+content,'w="fill" h="fill" gap={0} items="stretch"'),
 `name="Main window / ${active}" w={1040} h={704} gap={0} bg=${C('Surface',dark)} rounded=${token('Radius/Window')} overflow="hidden" shadow="0 16 48 #1B243314"`);
}
function pageTitle(title,actions=''){return row(text(title,24,'Text',600)+spacer()+actions,'w="fill" h={44}');}
function field(label,value,opts={}){
 return row(text(label,13,'Secondary',400,'w={104}')+
 col(row(text(value,14,opts.placeholder?'Muted':'Text',400,'w="fill"')+(opts.select?icon('chevron','Muted',false,14):''),'w="fill" h={34}')+rule(),'w="fill" gap={0}'),
 'w="fill" h={42} gap={16}');
}
function sectionTitle(s,extra=''){return row(text(s,14,'Text',600)+spacer()+extra,'w="fill" h={32}');}
function settings(){
 const content=col(pageTitle('设置')+
 row(text('模型',14,'Accent',600)+text('翻译',14,'Secondary')+text('快捷键',14,'Secondary')+text('外观',14,'Secondary'),'w="fill" gap={24} h={40}')+
 rule()+
 col(sectionTitle('连接')+field('接口地址','https://api.deepseek.com')+field('模型','deepseek-chat')+field('API Key','••••••••••••••••••••••••'),'gap={4} w="fill"')+
 col(sectionTitle('生成参数')+field('思考等级','关闭',{select:true})+field('温度','0.2')+field('输出上限','使用模型默认值',{placeholder:true})+button('更多参数'),'gap={4} w="fill"')+
 frame('','w={1} h="fill"')+rule()+
 row(button('保存')+button('测试连接')+spacer()+row(icon('check','Success',false,14)+text('连接正常',12,'Success')),'w="fill" h={40}'),
 'w="fill" h="fill" px={28} pt={12} pb={16} gap={12}');
 return shell('设置',content);
}
function translate(){
 const left=col(row(text('自动识别',13,'Secondary',500)+spacer()+button('清空'),'w="fill"')+
 text('The real voyage of discovery consists not in seeking new landscapes, but in having new eyes.',20,'Text',400,'w="fill"')+
 frame('','w={1} h="fill"')+row(button('粘贴')+spacer()+text('107 / 12,000',12,'Muted'),'w="fill"'),
 'w="fill" h="fill" gap={20}');
 const right=col(row(text('简体中文',13,'Secondary',500)+spacer()+button('','refresh'),'w="fill"')+
 text('真正的发现之旅，不在于寻找新的风景，而在于拥有新的眼光。',20,'Text',400,'w="fill"')+
 frame('','w={1} h="fill"')+row(button('复制','copy')+spacer(),'w="fill"'),'w="fill" h="fill" gap={20}');
 const content=col(pageTitle('翻译',button('演示模式'))+
 row(text('自动',13,'Accent',600)+text('单词',13,'Secondary')+text('句子',13,'Secondary')+spacer()+button('翻译','arrow'),'w="fill" gap={20} h={40}')+
 rule()+row(left+frame('',`w={1} h="fill" bg=${C('Rule')}`)+right,'w="fill" h="fill" pt={12} gap={24}')+
 rule()+row(text('DeepSeek · deepseek-chat',12,'Secondary')+spacer()+text('Ctrl + Enter',12,'Muted'),'w="fill" h={28}'),
 'w="fill" h="fill" px={28} pt={12} pb={16} gap={12}');
 return shell('翻译',content);
}
function vocabulary(){
 const words=[['bank','银行；金融机构','今天'],['serendipity','意外发现的美好','昨天'],['nuance','细微差别','昨天'],['resilient','有韧性的；适应力强的','10 月 18 日']];
 const list=col(row(icon('search','Muted',false,16)+text('搜索词汇',13,'Muted'),'w="fill" h={36}')+rule()+
 words.map(([w,m,d],i)=>row(frame('',`w={3} h={36} rounded=${token('Radius/Indicator')} ${i===0?`bg=${C('Indicator')}`:''}`)+col(row(text(w,16,i===0?'SelectedText':'Text',600)+spacer()+text(d,12,i===0?'SelectedSecondary':'Muted'),'w="fill"')+text(m,13,i===0?'SelectedSecondary':'Secondary','400','w="fill"'),'w="fill" gap={4}'),`w="fill" pl={5} pr={12} py={12} gap={8} rounded=${token('Radius/Row')} ${i===0?`bg=${C('Selected')}`:''}`)).join(''),
 'w={264} h="fill" gap={4}');
 const detail=col(row(text('bank',28,'Text',600)+spacer()+button('','more'),'w="fill"')+
 row(text('英语 → 简体中文',12,'Secondary')+row(icon('check','Success',false,14)+text('已收藏',12,'Success'),'gap={4}'),'gap={12}')+
 col(text('名词',12,'Secondary')+text('银行；金融机构',18),'gap={4} w="fill" pt={16}')+
 col(text('笔记',13,'Secondary',600)+text('添加你的理解或记忆线索',14,'Muted')+rule(),'w="fill" gap={12} pt={24}')+
 frame('','w={1} h="fill"')+row(button('复制','copy')+spacer()+button('移除'),'w="fill"'),
 'w="fill" h="fill" gap={8} pl={24}');
 return shell('词汇表',col(pageTitle('词汇表',button('导出','download'))+
 text('4 个词汇',12,'Secondary')+rule()+row(list+frame('',`w={1} h="fill" bg=${C('Rule')}`)+detail,'w="fill" h="fill" gap={16}'),
 'w="fill" h="fill" px={28} pt={12} pb={24} gap={12}'));
}
function settingsGeneral(){
 return shell('设置',col(pageTitle('设置')+row(text('模型',14,'Secondary')+text('翻译',14,'Accent',600)+text('快捷键',14,'Secondary')+text('外观',14,'Secondary'),'w="fill" gap={24} h={40}')+rule()+
 col(sectionTitle('语言')+field('首选语言','简体中文',{select:true})+field('第二语言','English',{select:true}),'w="fill" gap={4}')+
 col(sectionTitle('划词')+row(checkbox('自动划词',true)+spacer()+text('选中文字时显示工具条',12,'Secondary'),'w="fill"')+
 row(checkbox('附带上下文',true)+spacer()+text('帮助判断单词含义',12,'Secondary'),'w="fill"')+row(checkbox('开机启动',false)+spacer(),'w="fill"'),'w="fill" gap={4}')+
 col(sectionTitle('排除应用',button('添加'))+text('未添加',13,'Muted'),'w="fill" gap={4}')+
 frame('','h="fill" w={1}')+rule()+row(button('保存')+spacer(),'w="fill" h={40}'),
 'w="fill" h="fill" px={28} pt={12} pb={16} gap={12}'));
}
const overview=board('01 · Floating translation',
 caption('SQUIRREL / DESIGN REVIEW 01','轻巧地出现，清楚地表达。','翻译浮窗 · 光感中性色 · 只保留阅读与必要操作')+
 row(labelled('划词工具条 · 40 高',toolbar())+spacer()+text('无系统边框 / 12px 圆角 / 柔和投影',12,'Secondary'),'w="fill"')+
 row(labelled('单词 · 440 × 224',popup('word'))+labelled('句子 · 536 × 208',popup('sentence')),'gap={40} items="start"')+
 row(labelled('深色 · 同一层级与间距',popup('word',true))+labelled('流式等待 · 单行思考，正文开始后隐藏',popup('loading')),'gap={40} items="start"')
);
const boards=[
 ['floating',overview,0,0],
 ['translate',board('02 · Translation workspace',caption('SQUIRREL / DESIGN REVIEW 02','翻译工作台','稳定双栏 · 输入与结果并置 · 无边框窗口')+translate()),1240,0],
 ['vocabulary',board('03 · Vocabulary',caption('SQUIRREL / DESIGN REVIEW 03','词汇表','列表与详情分工 · 紧凑释义 · 笔记留给自己')+vocabulary()),2480,0],
 ['settings',board('04 · Model settings',caption('SQUIRREL / DESIGN REVIEW 04','配置有秩序，阅读有重点。','模型设置 · 固定标签列 · 逐层展开参数')+settings()),0,1120],
 ['preferences',board('05 · Translation settings',caption('SQUIRREL / DESIGN REVIEW 05','行为与语言','复选项统一对齐 · 辅助说明仅在需要时出现')+settingsGeneral()),1240,1120],
 ['states',board('06 · Resilience and states',caption('SQUIRREL / DESIGN REVIEW 06','状态也属于设计','长文 / 错误 / 深色 / 交互反馈')+
 row(labelled('长文 · 仅正文滚动',popup('long'))+col(labelled('错误 · 明确恢复入口',popup('error'))+labelled('深色句子',popup('sentence',true)),'gap={24}'),'gap={32} items="start"'),1176),2480,1120]
];
writeFileSync('artifacts/openpencil/render.json',JSON.stringify(boards.map(([key,jsx,x,y])=>{
 writeFileSync(`design/${key}.jsx`,jsx);
 return {name:'render',args:{...target,jsx,x,y},output:`artifacts/openpencil/${key}-result.json`};
})));
// Component masters use native identity; intended for subsequent UI implementation.
const comp=board('07 · Foundation & components',caption('SQUIRREL / DESIGN SYSTEM','以文字与空间构成界面','4px 基准 / 语义色 / 固定点击区 / 状态不改变布局')+
 row(col(text('Typography',12,'Accent',600)+text('标题 · Settings',24,'Text',600)+text('正文用于阅读与理解。',16)+text('标签与操作',14,'Secondary')+text('辅助信息 / Caption',12,'Secondary'),'w={420} gap={12}')+
 col(text('Semantic palette',12,'Accent',600)+['Surface','Sidebar','Text','Secondary','Accent','Hover','Selected','SelectedText','Success','Danger'].map(n=>row(frame('',`w={24} h={24} rounded={6} bg=${C(n)}`)+text(n,13,'Secondary')+spacer()+text(colors['Light/'+n],12,'Secondary'),'w="fill"')).join(''),'w={420} gap={8}'),'gap={64} items="start"')+
 rule()+text('Ghost controls · 每个状态保持同一尺寸',14,'Text',600)+
 row(['Default','Hover','Focus','Disabled'].map((state,i)=>col(text(state,12,'Secondary')+
 `<Component name="Copy / State=${state}" w={88} h={32} flex="row" items="center" justify="center" gap={8} rounded=${token('Radius/Control')} ${i===1?`bg=${C('Hover')}`:''} ${i===2?`stroke=${C('Focus')} strokeWidth={2}`:''} ${i===3?'opacity={0.4}':''}>${icon('copy')}${text('复制',13,'Secondary',500)}</Component>`,'gap={12}')).join(''),'gap={48}')+
 row(checkbox('自动划词',true)+checkbox('开机启动',false)+button('简体中文'),'gap={40}')+
 rule()+text('Spacing / 4 · 8 · 12 · 16 · 24 · 32',14,'Text',600)+
 text('紧密关联 4–8；组内 12–16；跨区 24–32。浮窗正文 16，主内容 28，标签列 104。',13,'Secondary')+
 text('圆角：窗口 12；按钮、菜单选中和列表 8；复选框 4；选中标记 2。各交互状态不改变圆角与内距。',13,'Secondary')
);
writeFileSync('design/foundations.jsx',comp);
writeFileSync('artifacts/openpencil/components.json',JSON.stringify([{name:'render',args:{...target,jsx:comp,x:0,y:2240},output:'artifacts/openpencil/foundations-result.json'}]));
function controlState(label,ico,tone,state,dark=false){
 const bg=state==='Hover'?(tone==='Accent'?'AccentHover':tone==='Danger'?'DangerHover':'Hover'):
 state==='Pressed'?(tone==='Accent'?'AccentPressed':tone==='Danger'?'DangerPressed':'Pressed'):null;
 return row(icon(ico,tone,dark)+text(label,13,tone,400,'',dark),
 `name="${label} / ${state}" w={100} h={32} justify="center" rounded=${token('Radius/Control')} ${bg?`bg=${C(bg,dark)}`:''} ${state==='Focus'?`stroke=${C('Focus',dark)} strokeWidth={2}`:''} ${state==='Disabled'?'opacity={0.4}':''}`);
}
function stateMatrix(dark=false){
 return col(row(text(dark?'深色':'浅色',13,'Text',600,'w={132}',dark)+['Default','Hover','Pressed','Focus','Disabled'].map(s=>text(s,12,'Secondary',400,'w={100}',dark)).join(''),'gap={20}')+
 [['普通操作','复制','copy','Secondary'],['主要操作','翻译','translate','Accent'],['危险操作','移除','close','Danger']].map(([title,label,ico,tone])=>
 row(text(title,13,'Secondary',400,'w={132}',dark)+['Default','Hover','Pressed','Focus','Disabled'].map(state=>controlState(label,ico,tone,state,dark)).join(''),'gap={20}')).join(''),
 `w="fill" px={24} py={20} gap={12} bg=${C('Surface',dark)} rounded=${token('Radius/Window')}`);
}
const stateBoard=board('08 · Color and interaction states',
 caption('SQUIRREL / DESIGN REVIEW 02 · REVISION','颜色有分工，状态有边界。','中性操作 / 主操作 / 成功与危险 · 选中状态与 hover 分离')+
 row(labelled('浅色侧栏',col(navItem('翻译','translate')+navItem('词汇表','book',false,'Hover')+navItem('设置','settings',true),`w={184} p={12} gap={4} bg=${C('Sidebar')}`))+
 labelled('选中 + hover',col(navItem('翻译','translate')+navItem('词汇表','book')+navItem('设置','settings',true,'Hover'),`w={184} p={12} gap={4} bg=${C('Sidebar')}`))+
 labelled('深色侧栏',col(navItem('翻译','translate',false,'Default',true)+navItem('词汇表','book',false,'Hover',true)+navItem('设置','settings',true,'Default',true),`w={184} p={12} gap={4} bg=${C('Sidebar',true)}`))+
 col(text('3px 选中标记',14,'Text',600)+text('背景、字重、标记共同表达选中。\n普通 hover 使用中性灰，保持区别。',13,'Secondary','400','w="fill"')+text('40px 行高 / 8px 圆角',12,'Secondary'),'w="fill" gap={8}'),'w="fill" gap={28} items="start"')+
 stateMatrix()+stateMatrix(true)+
 row(row(icon('check','Success')+text('连接正常',13,'Success'))+row(icon('check','Success')+text('已收藏',13,'Success'))+spacer()+text('成功色仅用于明确的结果反馈',12,'Secondary'),'w="fill" gap={24}')+
 rule()+text('对齐规则',14,'Text',600)+
 text('浮窗标题、正文首字、左侧操作图标共用 x = 16；按钮点击区从 x = 8 开始，内部保留 8px。',13,'Secondary')+
 text('圆角跟随元素尺寸：窗口 12 / 32px 按钮与 40px 菜单 8 / 14px 复选框 4 / 3px 状态条 2。',13,'Secondary')
);
writeFileSync('design/interaction-states.jsx',stateBoard);
writeFileSync('artifacts/openpencil/interaction-states.json',JSON.stringify([{name:'render',args:{...target,jsx:stateBoard,x:1240,y:2240},output:'artifacts/openpencil/interaction-states-result.json'}]));
