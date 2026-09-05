#!/usr/bin/env python3
"""存量迁移：遗留 .archimate → .diff.archimate + 基于上版全量生成 .full.archimate

用法：在 docs 仓库根执行 python3 architecture/scripts/migrate-legacy-archimate.py
产出：<stem>.diff.archimate（原文件重命名）+ <stem>.full.archimate（合并生成）
"""
import re, os, sys, subprocess

ARCH = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..')
os.chdir(ARCH)

ELEM_RE = re.compile(r'<element xsi:type="archimate:([^"]+)" name="([^"]*)" id="([^"]+)"/>')
# 关系两种属性顺序：diff 风格 type→id→source→target；v65 base 风格 type→name→id→source→target
REL_RE = re.compile(r'<element xsi:type="archimate:([^"]+)"(?: name="([^"]*)")? id="([^"]+)" source="([^"]+)" target="([^"]+)"/>')

TYPE_FOLDER = {
    'Plateau': 'implementation_migration', 'Gap': 'implementation_migration',
    'WorkPackage': 'implementation_migration', 'Deliverable': 'implementation_migration',
}
def folder_type(t):
    if t in TYPE_FOLDER: return TYPE_FOLDER[t]
    if t.startswith('Business'): return 'business'
    if t.startswith('Application') or t == 'DataObject': return 'application'
    if t.startswith('Technology') or t in ('Artifact', 'SystemSoftware', 'Device', 'Node'): return 'technology'
    return 'application'

def view_row(t, name):
    if '[DEPRECATED' in name or '[RETIRED' in name: return 'dep'
    ft = folder_type(t)
    return {'implementation_migration': 'impl', 'business': 'biz', 'application': 'app',
            'technology': 'tech', 'application_data': 'app'}.get(ft, 'app') if t != 'ApplicationDataObject' else 'data'

def insert_element(xml, etype, ename, eid):
    """插入新元素到对应 layer folder（folder 平铺、按 type 定位）"""
    ft = folder_type(etype)
    m = re.search(r'<folder[^>]*type="' + ft + r'"[^>]*>(.*?)(</folder>)', xml, re.S)
    if m:
        tag = f'  <element xsi:type="archimate:{etype}" name="{ename}" id="{eid}"/>'
        return xml[:m.end(1)] + '\n' + tag + xml[m.end(1):]
    # folder 不存在 → 在 Relations folder 前新建
    m = re.search(r'<folder[^>]*type="relations"[^>]*>', xml)
    if not m:
        raise RuntimeError(f'no relations folder for {eid}')
    name = {'business': 'Business', 'application': 'Application', 'technology': 'Technology',
            'implementation_migration': 'Implementation'}[ft]
    tag = (f'  <folder name="{name}" id="f-{ft}-auto" type="{ft}">\n'
           f'    <element xsi:type="archimate:{etype}" name="{ename}" id="{eid}"/>\n  </folder>\n')
    return xml[:m.start()] + tag + xml[m.start():]

def build_full_view(elems, rels, version, title, base_xml):
    """生成全量拓扑视图：所有元素网格布局 + 全部关系连线"""
    ROW_Y = {'impl': 40, 'biz': 150, 'app': 260, 'data': 380, 'tech': 500, 'dep': 620}
    rows = {'impl': [], 'biz': [], 'app': [], 'data': [], 'tech': [], 'dep': []}
    for eid, etype, ename in elems:
        if etype == 'ArchiMateDiagramModel':
            continue
        rows[view_row(etype, ename)].append((eid, etype, ename))
    do_id = {}
    for row, items in rows.items():
        for k, (eid, etype, ename) in enumerate(items):
            do_id[eid] = f'do-f-{eid}'
    do_meta = {}
    for row, items in rows.items():
        for k, (eid, etype, ename) in enumerate(items):
            do_meta[eid] = (k * 240, ROW_Y[row])
    # 连线：每个 relation → sourceConnection + targetConnections
    conns_of = {}          # source_eid -> [conn_xml]
    target_list = {}       # target_eid -> [conn_id]
    for rtype, rid, src, tgt in rels:
        if src not in do_id or tgt not in do_id:
            continue
        cid = f'cf-{rid}'
        conns_of.setdefault(src, []).append(
            f'<sourceConnection xsi:type="archimate:Connection" id="{cid}" '
            f'source="{do_id[src]}" target="{do_id[tgt]}" archimateRelationship="{rid}"/>')
        target_list.setdefault(tgt, []).append(cid)
    # 视图 XML（行内元素按行顺序输出）
    order = []
    for row in ('impl', 'biz', 'app', 'data', 'tech', 'dep'):
        for eid, etype, ename in rows[row]:
            order.append((row, eid, etype, ename))
    lines = []
    for row, eid, etype, ename in order:
        x, y = do_meta[eid]
        conns = '\n'.join(conns_of.get(eid, []))
        tg = f' targetConnections="{" ".join(target_list.get(eid, []))}"' if target_list.get(eid) else ''
        lines.append(f'      <child xsi:type="archimate:DiagramObject" id="{do_id[eid]}" archimateElement="{eid}"{tg}>')
        lines.append(f'        <bounds x="{x}" y="{y}" width="220" height="55"/>')
        if conns:
            lines.append(conns)
        lines.append('      </child>')
    body = '\n'.join(lines)
    # 视图类型拼写须跟随 base：Archi 不接受同一模型混用
    # ArchimateDiagramModel(MCP 拼写) 与 ArchiMateDiagramModel(标准拼写)
    diag = ('archimate:ArchimateDiagramModel' if 'ArchimateDiagramModel' in base_xml
            else 'archimate:ArchiMateDiagramModel')
    return (f'    <element xsi:type="{diag}" name="v{version} 全量拓扑 — {title}" id="view-full-{version}">\n'
            f'{body}\n    </element>')

# diff 为 Open Group 3.0 命名空间（类型名带层前缀），base 若为 archimatetool.com
# 命名空间（v65 风格：DataObject/Node 无前缀）则须映射，否则 Archi 报 incompatible
TYPE_MAP = {
    'ApplicationDataObject': 'DataObject',
    'TechnologyNode': 'Node',
}
def norm_type(base_xml, t):
    if 'http://www.opengroup.org/xsd/archimate/3.0/' not in base_xml:
        return TYPE_MAP.get(t, t)
    return t

def merge_full(base_xml, diff_xml, version, title, stem):
    """上版全量 + 本版增量 → 本版全量"""
    out = base_xml
    b_elem_ids = {eid: (t, n) for t, n, eid in ELEM_RE.findall(base_xml)}
    b_rel_ids = {rid: (t, n, s, g) for t, n, rid, s, g in REL_RE.findall(base_xml)}
    stats = {'updated': 0, 'inserted': [], 'rel_updated': 0, 'rel_kept': 0, 'rel_new': []}
    # 元素：id 匹配 → 替换 name/type；否则插入（类型名须适配 base 命名空间）
    for etype, ename, eid in ELEM_RE.findall(diff_xml):
        etype = norm_type(base_xml, etype)
        if eid in b_elem_ids:
            pat = re.compile(r'<element xsi:type="archimate:[^"]+" name="[^"]*" id="' + re.escape(eid) + r'"/>')
            new_tag = f'<element xsi:type="archimate:{etype}" name="{ename}" id="{eid}"/>'
            out, n = pat.subn(new_tag, out)
            assert n == 1, f'element {eid} sub failed'
            stats['updated'] += 1
        else:
            out = insert_element(out, etype, ename, eid)
            stats['inserted'].append(eid)
    # 关系：id 匹配 → 端点/类型未变则保留 base 原 tag（保 name），变更则重写；否则追加到 Relations folder
    for rtype, rname, rid, src, tgt in REL_RE.findall(diff_xml):
        if rid in b_rel_ids:
            bt, bname, bs, bg = b_rel_ids[rid]
            if bt == rtype and bs == src and bg == tgt:
                stats['rel_kept'] += 1
                continue
            pat = re.compile(r'<element xsi:type="archimate:[^"]+"(?: name="[^"]*")? id="'
                             + re.escape(rid) + r'" source="[^"]+" target="[^"]+"/>')
            if bname:
                new_tag = f'<element xsi:type="archimate:{rtype}" name="{bname}" id="{rid}" source="{src}" target="{tgt}"/>'
            else:
                new_tag = f'<element xsi:type="archimate:{rtype}" id="{rid}" source="{src}" target="{tgt}"/>'
            out, n = pat.subn(new_tag, out)
            assert n == 1, f'rel {rid} sub failed'
            stats['rel_updated'] += 1
        else:
            m = re.search(r'<folder[^>]*type="relations"[^>]*>(.*?)(</folder>)', out, re.S)
            tag = f'  <element xsi:type="archimate:{rtype}" id="{rid}" source="{src}" target="{tgt}"/>'
            out = out[:m.end(1)] + '\n' + tag + out[m.end(1):]
            stats['rel_new'].append(rid)
    # 全量拓扑视图（插入 Views folder 末尾）
    elems = [(eid, t, n) for t, n, eid in ELEM_RE.findall(out)]
    rels = [(t, rid, s, g) for t, _, rid, s, g in REL_RE.findall(out)]
    view = build_full_view(elems, rels, version, title, base_xml)
    m = re.search(r'<folder[^>]*type="diagrams"[^>]*>(.*?)(</folder>)', out, re.S)
    if not m:
        m = re.search(r'(</archimate:model>)', out)
        view = '\n  <folder name="Views" id="f-views" type="diagrams">\n' + view + '\n  </folder>\n' + m.group(1)
        out = out[:m.start(1)] + view
    else:
        out = out[:m.end(1)] + '\n' + view + out[m.end(1):]
    # 模型元数据：identifier（须 ASCII——xs:ID 非法字符会导致 Archi GUI 报
    # "Feature 'identifier' not found"）/ name / purpose
    mdate = re.search(r'-(\d{8})-', stem)
    new_id = f'model-v{version}-full-{mdate.group(1)}' if mdate else f'model-v{version}-full'
    out = re.sub(r'identifier="[^"]*"', f'identifier="{new_id}"', out, count=1)
    out = re.sub(r'<name xml:lang="en">[^<]*</name>', f'<name xml:lang="en">v{version} 全量 — {title} (post-change)</name>', out, count=1)
    m = re.search(r'<purpose>(.*?)</purpose>', out, re.S)
    if m:
        out = out.replace(m.group(0), m.group(0)[:-len('</purpose>')] +
                          f'\nv{version} 全量模型：合并「{title}」变更后的完整架构。' + '</purpose>')
    return out, stats

JOBS = [
    # (diff_stem, base_full, version, 迭代名)
    ('v66-application-integration-20260806-0040', 'v65-application-integration-20260805-2347-claude.archimate', 66, '多会话互知'),
    ('v67-application-integration-20260807-1920', 'v66-application-integration-20260806-0040-claude.full.archimate', 67, '任务帖存续期'),
    ('v68-application-integration-20260808-1732', 'v67-application-integration-20260807-1920-claude.full.archimate', 68, '插件 OIDC 白名单管理'),
    ('v13-enterprise-landscape-20260806-0008', 'v12-enterprise-landscape-20260805-1800-claude.archimate', 13, 'Git Hooks 入库'),
    ('v14-enterprise-landscape-20260806-0040', 'v13-enterprise-landscape-20260806-0008-claude.full.archimate', 14, '多会话互知'),
    ('v15-enterprise-landscape-20260807-1920', 'v14-enterprise-landscape-20260806-0040-claude.full.archimate', 15, '任务帖存续期'),
    ('v16-enterprise-landscape-20260808-1732', 'v15-enterprise-landscape-20260807-1920-claude.full.archimate', 16, '插件 OIDC 白名单管理'),
]

if __name__ == '__main__':
    for diff_stem, base_full, version, title in JOBS:
        diff = diff_stem + '-claude.diff.archimate'
        base = base_full
        out_full = diff_stem + '-claude.full.archimate'
        if not os.path.exists(base):
            print(f'❌ 缺 base: {base}'); sys.exit(1)
        if not os.path.exists(diff):
            print(f'❌ 缺 diff: {diff}'); sys.exit(1)
        base_xml = open(base).read()
        diff_xml = open(diff).read()
        merged, stats = merge_full(base_xml, diff_xml, version, title, diff_stem)
        with open(out_full, 'w') as f:
            f.write(merged)
        print(f'✅ {out_full}: 元素更新 {stats["updated"]}, 新增 {stats["inserted"]}, '
              f'关系保持 {stats["rel_kept"]}, 更新 {stats["rel_updated"]}, 新增 {stats["rel_new"]}')
