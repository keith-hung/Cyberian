"""Build the Power Automate legacy import package (calendar-export-flow.zip).

One flow, two files:
  calendar.json          snapshot of -7..+30 days with full fields, overwritten every run
  calendar-history.json  slim fields, -365 days..today, merged about once a day
"""
import json
import os
import zipfile

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "calendar-export-flow.zip")
NAME = "行事曆匯出"
SNAP_PATH = "/Calendar-Export/calendar.json"
HIST_PATH = "/Calendar-Export/calendar-history.json"
TZ = "Taipei Standard Time"
HOURS_20 = 20 * 3600 * 10_000_000  # ticks

# Fixed ids keep rebuilds reproducible.
FLOW_ID = "6c1f4a52-0d3b-4b8e-9a51-3f2e7c9d1a01"
API_O365 = "6c1f4a52-0d3b-4b8e-9a51-3f2e7c9d1a02"
CONN_O365 = "6c1f4a52-0d3b-4b8e-9a51-3f2e7c9d1a03"
API_OD = "6c1f4a52-0d3b-4b8e-9a51-3f2e7c9d1a04"
CONN_OD = "6c1f4a52-0d3b-4b8e-9a51-3f2e7c9d1a05"

O365 = {"apiId": "/providers/Microsoft.PowerApps/apis/shared_office365", "connectionName": "shared_office365"}
OD = {"apiId": "/providers/Microsoft.PowerApps/apis/shared_onedriveforbusiness", "connectionName": "shared_onedriveforbusiness"}
SECURE_IO = {"secureData": {"properties": ["inputs", "outputs"]}}
SECURE_IN = {"secureData": {"properties": ["inputs"]}}

SLIM_SELECT = "id,iCalUId,subject,start,end,isAllDay,location,organizer,isOnlineMeeting,showAs,isCancelled,sensitivity,type"
FULL_SELECT = SLIM_SELECT + ",bodyPreview,attendees,onlineMeeting,onlineMeetingUrl"


def ok(*names):
    return {n: ["Succeeded"] for n in names}


def api(host, op, params, after=None, secure=None):
    a = {"runAfter": after or {}, "type": "OpenApiConnection",
         "inputs": {"parameters": params, "host": dict(host, operationId=op),
                    "authentication": "@parameters('$authentication')"}}
    if secure:
        a["runtimeConfiguration"] = secure
    return a


def compose(expr, after, secure=False):
    a = {"runAfter": after, "type": "Compose", "inputs": expr}
    if secure:
        a["runtimeConfiguration"] = SECURE_IN
    return a


def select(src, mapping, after, secure=False):
    a = {"runAfter": after, "type": "Select", "inputs": {"from": src, "select": mapping}}
    if secure:
        a["runtimeConfiguration"] = SECURE_IO
    return a


def query(src, where, after, secure=False):
    a = {"runAfter": after, "type": "Query", "inputs": {"from": src, "where": where}}
    if secure:
        a["runtimeConfiguration"] = SECURE_IO
    return a


def terminate(msg):
    return {"runAfter": {}, "type": "Terminate", "inputs": {"runStatus": "Failed", "runError": {"message": msg}}}


def fail_if(name, expr, msg, after):
    return {"runAfter": after, "type": "If", "expression": expr,
            "actions": {f"Stop_{name}": terminate(msg)}, "else": {"actions": {}}}


def day(offset):
    return f"concat(convertFromUtc(addDays(utcNow(),{offset}),'{TZ}','yyyy-MM-dd'),'T00:00:00+08:00')"


def view_url(start, end, fields):
    return (f"concat('https://graph.microsoft.com/v1.0/me/calendar/calendarView?startDateTime=', "
            f"encodeUriComponent(outputs('{start}')), '&endDateTime=', encodeUriComponent(outputs('{end}')), "
            f"'&$select={fields}&$orderby=start/dateTime&$top=1000')")


def get_page(uri, after):
    return api(O365, "HttpRequest", {"Uri": uri, "Method": "GET", "CustomHeader1": f'Prefer: outlook.timezone="{TZ}"'},
               after, SECURE_IO)


SLIM_MAP = {
    "key": "@coalesce(item()?['iCalUId'], item()?['id'])",
    "subject": "@item()?['subject']",
    "start": "@concat(substring(item()?['start']?['dateTime'],0,19),'+08:00')",
    "end": "@concat(substring(item()?['end']?['dateTime'],0,19),'+08:00')",
    "tz": "@item()?['start']?['timeZone']",
    "allDay": "@item()?['isAllDay']",
    "location": "@item()?['location']?['displayName']",
    "organizer": "@item()?['organizer']?['emailAddress']?['name']",
    "online": "@item()?['isOnlineMeeting']",
    "showAs": "@item()?['showAs']",
    "status": "@if(equals(item()?['isCancelled'], true), 'cancelled', 'active')",
    "sensitivity": "@item()?['sensitivity']",
    "type": "@item()?['type']",
}
FULL_MAP = dict(SLIM_MAP, **{
    "preview": "@item()?['bodyPreview']",
    "attendees": "@item()?['attendees']",
    "meetingUrl": "@coalesce(item()?['onlineMeeting']?['joinUrl'], item()?['onlineMeetingUrl'], '')",
})


def bad_tz_guard(src, prefix, after):
    return {
        f"{prefix}BadTz": query(src, f"@not(equals(item()?['tz'], '{TZ}'))", after, secure=True),
        f"{prefix}TzGuard": fail_if(f"{prefix}TzGuard", {"and": [{"greater": [f"@length(body('{prefix}BadTz'))", 0]}]},
                                    "Prefer outlook.timezone header not applied; file not written", ok(f"{prefix}BadTz")),
    }


def file_body(events_expr, start, end, mode):
    return {"schema": "calendar", "version": 3, "kind": mode, "generatedAt": "@outputs('RunAt')",
            "windowStart": f"@outputs('{start}')", "windowEnd": f"@outputs('{end}')",
            "count": f"@length({events_expr})", "events": f"@{events_expr}"}


# ── snapshot: every run, one request, nothing sensitive in run history
snapshot = {
    "SnapStart": compose("@" + day(-7), ok("RunAt")),
    "SnapEnd": compose("@" + day(31), ok("SnapStart")),
    "GetSnap": get_page("@" + view_url("SnapStart", "SnapEnd", FULL_SELECT), ok("SnapEnd")),
    "SnapOverflow": fail_if("SnapOverflow", {"and": [{"not": {"equals": ["@empty(body('GetSnap')?['@odata.nextLink'])", True]}}]},
                            "More than 1000 events in snapshot window; file not written", ok("GetSnap")),
    "NormalizeSnap": select("@body('GetSnap')?['value']", FULL_MAP, ok("SnapOverflow"), secure=True),
    **bad_tz_guard("@body('NormalizeSnap')", "Snap", ok("NormalizeSnap")),
    "BuildSnap": compose(file_body("body('NormalizeSnap')", "SnapStart", "SnapEnd", "snapshot"), ok("SnapTzGuard"), secure=True),
    "SnapMeta": api(OD, "GetFileMetadataByPath", {"path": SNAP_PATH}, ok("BuildSnap")),
    "WriteSnap": api(OD, "UpdateFile", {"id": "@body('SnapMeta')?['Id']", "body": "@outputs('BuildSnap')"},
                     ok("SnapMeta"), SECURE_IN),
}

# ── history: slim fields, merged when the file is older than 20 hours (or still the starter file)
history_branch = {
    "ReadHist": api(OD, "GetFileContentByPath", {"path": HIST_PATH, "inferContentType": True}),
    "HistObj": compose("@if(equals(body('ReadHist')?['$content'], null), body('ReadHist'), "
                       "json(base64ToString(body('ReadHist')?['$content'])))", ok("ReadHist")),
    "Existing": compose("@coalesce(outputs('HistObj')?['events'], json('[]'))", ok("HistObj")),
    "ParseGuard": fail_if("ParseGuard", {"and": [{"equals": ["@length(outputs('Existing'))", 0]},
                                   {"greater": ["@body('HistMeta')?['Size']", 1024]}]},
                          "History file is not empty but parsed 0 events; file not written", ok("Existing")),
    "HistStart": compose("@" + day(-365), ok("ParseGuard")),
    "HistEnd": compose("@" + day(1), ok("HistStart")),
    "Set_histNext": {"runAfter": ok("HistEnd"), "type": "SetVariable",
                     "inputs": {"name": "histNext", "value": "@" + view_url("HistStart", "HistEnd", SLIM_SELECT)}},
    "Do_until": {
        "runAfter": ok("Set_histNext"), "type": "Until",
        "expression": "@empty(variables('histNext'))",
        "limit": {"count": 60, "timeout": "PT1H"},
        "actions": {
            "GetHistPage": get_page("@variables('histNext')", {}),
            "MergeHist": compose("@union(variables('histEvents'), body('GetHistPage')?['value'])", ok("GetHistPage")),
            "Set_histEvents": {"runAfter": ok("MergeHist"), "type": "SetVariable",
                               "inputs": {"name": "histEvents", "value": "@outputs('MergeHist')"}},
            "Set_histNextPage": {"runAfter": ok("Set_histEvents"), "type": "SetVariable",
                                 "inputs": {"name": "histNext", "value": "@coalesce(body('GetHistPage')?['@odata.nextLink'], '')"}},
        },
    },
    "NormalizeHist": select("@variables('histEvents')", SLIM_MAP, ok("Do_until")),
    "EmptyGuard": fail_if("EmptyGuard", {"and": [{"equals": ["@length(body('NormalizeHist'))", 0]}]},
                          "calendarView returned 0 events for the past year; file not written", ok("NormalizeHist")),
    **bad_tz_guard("@body('NormalizeHist')", "Hist", ok("EmptyGuard")),
    "FreshKeys": select("@body('NormalizeHist')", "@item()?['key']", ok("HistTzGuard")),
    "KeepOutside": query("@outputs('Existing')",
                         "@and(not(contains(body('FreshKeys'), item()?['key'])), or(less(ticks(item()?['start']), ticks(outputs('HistStart'))), greaterOrEquals(ticks(item()?['start']), ticks(outputs('HistEnd')))))",
                         ok("FreshKeys")),
    "Vanished": query("@outputs('Existing')",
                      "@and(not(contains(body('FreshKeys'), item()?['key'])), greaterOrEquals(ticks(item()?['start']), ticks(outputs('HistStart'))), less(ticks(item()?['start']), ticks(outputs('HistEnd'))))",
                      ok("KeepOutside")),
    "MarkDeleted": select("@body('Vanished')",
                          "@if(equals(item()?['status'],'active'), setProperty(setProperty(item(),'status','deleted'),'deletedDetectedAt', outputs('RunAt')), item())",
                          ok("Vanished")),
    "Combined": compose("@union(body('KeepOutside'), body('MarkDeleted'), body('NormalizeHist'))", ok("MarkDeleted")),
    "BuildHist": compose(file_body("outputs('Combined')", "HistStart", "HistEnd", "history"), ok("Combined")),
    "WriteHist": api(OD, "UpdateFile", {"id": "@body('HistMeta')?['Id']", "body": "@outputs('BuildHist')"}, ok("BuildHist")),
}

history = {
    "HistMeta": api(OD, "GetFileMetadataByPath", {"path": HIST_PATH}, ok("WriteSnap")),
    "HistoryDue": {
        "runAfter": ok("HistMeta"), "type": "If",
        "expression": {"or": [
            {"less": ["@body('HistMeta')?['Size']", 1024]},
            {"greater": [f"@sub(ticks(utcNow()), ticks(body('HistMeta')?['LastModified']))", HOURS_20]},
        ]},
        "actions": history_branch,
        "else": {"actions": {}},
    },
}

actions = {
    "RunAt": compose(f"@concat(convertFromUtc(utcNow(),'{TZ}','yyyy-MM-ddTHH:mm:ss'),'+08:00')", {}),
    "Init_histNext": {"runAfter": ok("RunAt"), "type": "InitializeVariable",
                      "inputs": {"variables": [{"name": "histNext", "type": "string", "value": ""}]}},
    "Init_histEvents": {"runAfter": ok("Init_histNext"), "type": "InitializeVariable",
                        "inputs": {"variables": [{"name": "histEvents", "type": "array", "value": []}]}},
}
snapshot["SnapStart"]["runAfter"] = ok("Init_histEvents")
actions.update(snapshot)
actions.update(history)

definition = {
    "name": FLOW_ID,
    "id": f"/providers/Microsoft.Flow/flows/{FLOW_ID}",
    "type": "Microsoft.Flow/flows",
    "properties": {
        "apiId": "/providers/Microsoft.PowerApps/apis/shared_logicflows",
        "displayName": NAME,
        "definition": {
            "$schema": "https://schema.management.azure.com/providers/Microsoft.Logic/schemas/2016-06-01/workflowdefinition.json#",
            "contentVersion": "1.0.0.0",
            "parameters": {"$authentication": {"defaultValue": {}, "type": "SecureObject"},
                           "$connections": {"defaultValue": {}, "type": "Object"}},
            "triggers": {"Recurrence": {
                "type": "Recurrence",
                "recurrence": {
                    "frequency": "Week", "interval": 1, "timeZone": TZ, "startTime": "2026-11-05T00:00:00",
                    "schedule": {"weekDays": ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"],
                                 "hours": list(range(24)), "minutes": [0, 30]},
                },
                "runtimeConfiguration": {"concurrency": {"runs": 1}},
            }},
            "actions": actions,
        },
        "connectionReferences": {
            "shared_office365": {"connectionName": "", "source": "Embedded",
                                 "id": "/providers/Microsoft.PowerApps/apis/shared_office365", "tier": "NotSpecified", "apiName": "office365"},
            "shared_onedriveforbusiness": {"connectionName": "", "source": "Embedded",
                                           "id": "/providers/Microsoft.PowerApps/apis/shared_onedriveforbusiness", "tier": "NotSpecified", "apiName": "onedriveforbusiness"},
        },
        "flowFailureAlertSubscribed": True,
        "isManaged": False,
    },
}


def api_res(name, disp):
    return {"id": f"/providers/Microsoft.PowerApps/apis/{name}", "name": name, "type": "Microsoft.PowerApps/apis",
            "suggestedCreationType": "Existing", "details": {"displayName": disp},
            "configurableBy": "System", "hierarchy": "Child", "dependsOn": []}


def conn_res(disp, api_id):
    return {"type": "Microsoft.PowerApps/apis/connections", "suggestedCreationType": "Existing",
            "creationType": "Existing", "details": {"displayName": disp},
            "configurableBy": "User", "hierarchy": "Child", "dependsOn": [api_id]}


manifest = {
    "schema": "1.0",
    "details": {"displayName": NAME, "description": "Export Outlook calendar to OneDrive Calendar-Export every 30 minutes",
                "createdTime": "2026-11-17T00:00:00Z", "packageTelemetryId": FLOW_ID,
                "creator": "N/A", "sourceEnvironment": ""},
    "resources": {
        FLOW_ID: {"type": "Microsoft.Flow/flows", "suggestedCreationType": "New", "creationType": "Existing, New, Update",
                  "details": {"displayName": NAME}, "configurableBy": "User", "hierarchy": "Root",
                  "dependsOn": [API_O365, CONN_O365, API_OD, CONN_OD]},
        API_O365: api_res("shared_office365", "Office 365 Outlook"),
        CONN_O365: conn_res("Office 365 Outlook", API_O365),
        API_OD: api_res("shared_onedriveforbusiness", "OneDrive for Business"),
        CONN_OD: conn_res("OneDrive for Business", API_OD),
    },
}

files = {
    "manifest.json": manifest,
    "Microsoft.Flow/flows/manifest.json": {"packageSchemaVersion": "1.0", "flowAssets": {"assetPaths": [FLOW_ID]}},
    f"Microsoft.Flow/flows/{FLOW_ID}/definition.json": definition,
    f"Microsoft.Flow/flows/{FLOW_ID}/apisMap.json": {"shared_office365": API_O365, "shared_onedriveforbusiness": API_OD},
    f"Microsoft.Flow/flows/{FLOW_ID}/connectionsMap.json": {"shared_office365": CONN_O365, "shared_onedriveforbusiness": CONN_OD},
}


def check_refs(acts, scope_names):
    """Every runAfter target must exist in the same scope."""
    for name, a in acts.items():
        for dep in a.get("runAfter", {}):
            assert dep in scope_names, f"{name} runs after unknown {dep}"
        for key in ("actions",):
            if key in a:
                check_refs(a[key], set(a[key]))
        if "else" in a:
            check_refs(a["else"]["actions"], set(a["else"]["actions"]))


def all_names(acts):
    for name, a in acts.items():
        yield name
        yield from all_names(a.get("actions", {}))
        yield from all_names(a.get("else", {}).get("actions", {}))


check_refs(actions, set(actions))
names = list(all_names(actions))
dupes = {n for n in names if names.count(n) > 1}
assert not dupes, f"duplicate action names: {dupes}"
with zipfile.ZipFile(OUT, "w", zipfile.ZIP_DEFLATED) as z:
    for path, obj in files.items():
        z.writestr(path, json.dumps(obj, ensure_ascii=False, indent=2))
print(os.path.normpath(OUT), os.path.getsize(OUT))
