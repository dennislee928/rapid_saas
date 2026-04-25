import { AppShell } from "@/components/app-shell";
import { RulesEditor } from "@/components/rules-editor";
import { activeRuleDocument } from "@/lib/mock-data";

export default function RulesPage() {
  return (
    <AppShell eyebrow="Rule engine" title="Edit the ordered chain before the canvas exists.">
      <RulesEditor initialRules={activeRuleDocument} />
    </AppShell>
  );
}
