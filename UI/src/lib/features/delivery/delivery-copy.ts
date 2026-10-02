const italian: Record<string, string> = {
	'Changes needed': 'Modifiche necessarie',
	'Testing blocked': 'Test bloccati',
	'No tests recorded': 'Nessun test registrato',
	'Testing pending': 'Test da completare',
	'Delivery work pending': 'Consegna da completare',
	'Ready for review': 'Pronto per la revisione',
	'Could not load the delivery plan. Try again.': 'Impossibile caricare il piano. Riprova.',
	'Leave this project and discard unsaved delivery changes?': 'Uscire dal progetto e perdere le modifiche non salvate?',
	'Someone saved a newer plan. Reload the latest version before editing again.':
		'Il piano è stato aggiornato. Ricaricalo prima di modificarlo.',
	'Give every milestone and test case a title before saving.':
		'Assegna un titolo a ogni milestone e test prima di salvare.',
	'Passed and failed tests need recorded evidence before saving.':
		'I test superati o falliti richiedono una prova prima di salvare.',
	'Delivery plan saved.': 'Piano salvato.',
	'Someone saved a newer plan. Your edits are still here. Copy anything you want to keep before reloading.':
		'Il piano è stato aggiornato. Le tue modifiche sono ancora qui: copiale prima di ricaricare.',
	'Could not save the delivery plan. Your edits are still here. Try again.':
		'Impossibile salvare il piano. Le tue modifiche sono ancora qui. Riprova.',
	'Loading delivery plan…': 'Caricamento del piano…',
	'Retry loading': 'Riprova',
	'Product delivery': 'Consegna prodotto',
	'Brief, milestones and manual verification.': 'Brief, milestone e verifica manuale.',
	'Unsaved changes': 'Modifiche non salvate',
	'Cancel changes': 'Annulla modifiche',
	'Saving…': 'Salvataggio…',
	'Save plan': 'Salva piano',
	'View only. Workspace owners, admins, and members can edit this plan.':
		'Sola lettura. Proprietari, amministratori e membri possono modificare il piano.',
	'Reload latest and discard my edits': 'Ricarica e scarta le mie modifiche',
	'Based on unsaved changes': 'Include modifiche non salvate',
	'Milestones done': 'Milestone completate',
	Passed: 'Superati',
	Failed: 'Falliti',
	Blocked: 'Bloccati',
	'Not run': 'Da eseguire',
	'How readiness works': 'Come funziona la verifica',
	'Ready for review requires a complete brief, finished milestones, and every test passed with steps, expected results and evidence. Release approval remains a separate decision.':
		'La revisione richiede un brief completo, milestone concluse e tutti i test superati con passaggi, risultati attesi e prove. Il rilascio richiede una decisione separata.',
	'Product brief': 'Brief prodotto',
	'Product name': 'Nome prodotto',
	'Target release': 'Rilascio previsto',
	Objective: 'Obiettivo',
	'Success metric': 'Criterio di successo',
	'What are you delivering?': 'Cosa stai realizzando?',
	'The problem this release should solve': 'Il problema da risolvere',
	'How will you know it worked?': 'Come verificherai il risultato?',
	Milestones: 'Milestone',
	'Add milestone': 'Aggiungi milestone',
	'No milestones yet.': 'Nessuna milestone.',
	'Due date': 'Scadenza',
	'Milestone status': 'Stato milestone',
	Planned: 'Pianificata',
	'In progress': 'In corso',
	Done: 'Completata',
	'Manual tests': 'Test manuali',
	'Add test case': 'Aggiungi test',
	'Record the result and evidence after each manual check.': 'Registra risultato e prova dopo ogni verifica manuale.',
	'Add a test case to assess readiness.': 'Aggiungi un test per verificare la consegna.',
	Steps: 'Passaggi',
	'Expected result': 'Risultato atteso',
	'Actions to perform': 'Azioni da eseguire',
	'What should happen': 'Cosa deve accadere',
	'Test status': 'Esito test',
	Evidence: 'Prova',
	'Observed result, environment, and links to proof': 'Risultato osservato, ambiente e link alla prova',
	'Test case': 'Test',
	'Start from a template:': 'Usa un modello:',
	Acceptance: 'Accettazione',
	Regression: 'Regressione',
	Accessibility: 'Accessibilità',
	'Search tests': 'Cerca test',
	'Filter by result': 'Filtra per esito',
	'All results': 'Tutti gli esiti',
	'No matching tests.': 'Nessun test corrispondente.',
	'Acceptance check': 'Verifica di accettazione',
	'Use the product as the intended user and complete the main workflow. Replace these steps with your release criteria.':
		'Usa il prodotto come utente previsto e completa il flusso principale. Sostituisci questi passaggi con i criteri del rilascio.',
	'The user completes the workflow and the success metric is met.':
		'L’utente completa il flusso e raggiunge il criterio di successo.',
	'Regression check': 'Verifica di regressione',
	'Repeat a previously working workflow affected by this release. Record the environment and test data.':
		'Ripeti un flusso già funzionante coinvolto dal rilascio. Registra ambiente e dati del test.',
	'Existing behavior remains correct and saved data is preserved.':
		'Il comportamento esistente resta corretto e i dati salvati vengono conservati.',
	'Accessibility check': 'Verifica di accessibilità',
	'Complete the main workflow using only the keyboard. Check visible focus, field labels and announcements with a screen reader.':
		'Completa il flusso principale con la tastiera. Verifica focus, etichette e annunci con un lettore di schermo.',
	'Every action is reachable, focus stays visible, and controls and errors have clear accessible names.':
		'Ogni azione è raggiungibile, il focus è visibile e controlli ed errori hanno nomi accessibili chiari.'
};
export function deliveryText(text: string, locale: string): string {
	return locale === 'it' ? (italian[text] ?? text) : text;
}
