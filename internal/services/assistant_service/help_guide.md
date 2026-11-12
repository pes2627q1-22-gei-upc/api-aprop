# Aprop user guide

Source: the Flutter app (frontend-aprop), checked 2026-11-07. The app UI is only in Catalan, so button and screen names are quoted exactly as shown.
Keep this file in sync with the app: the assistant answers only from it. Each "## key" is a topic of the get_app_help tool.

## overview
The app has 5 tabs at the bottom: "Inici" (home), "Agenda", "Descobreix", "Avisos" (notifications) and "Perfil".
"Inici" lists activities (list or map), has the "Assistent" button (robot icon) and the "Crear activitat" (+) button.
Not available yet: changing the app language or theme, voice input, searching people, editing or deleting activities.

## login-and-account
- Log in: screen "Inicia sessió": write the email and "Contrasenya" and tap "Entra". Or tap "Continua amb Google".
- Register: on the login screen tap "Crea un compte", fill Nom, Cognoms, username (3 to 20 characters), email and password, then "Crea el compte".
- After registering you must verify the email: screen "Verifica el teu correu", with the button "Torna a enviar el correu" if it did not arrive.
- With Google the first time you pick a username and tap "Continua".
- Log out: tab "Perfil" > gear icon ("Configuració") > "Tancar sessió".
- "Eliminar compte" appears in settings but is not available yet.

## find-activities
- The "Inici" tab lists activities. Above the list there are three buttons: "Catalunya", "Dates" and "Categories".
- "Dates" opens a calendar with shortcuts "Avui", "Demà" and "Aquest cap de setmana". Tap one day, or tap a second day to choose a range. Confirm with "Aplicar" and clear with "Netejar".
- "Categories" opens a list of checkboxes (an activity matches if it has any of the chosen categories). Confirm with "Aplicar".
- To remove all filters tap the "Netejar tots els filtres" icon.
- The "Catalunya" location filter is not available yet.
- You can also ask the assistant for activities in your own words.

## search
- Go to the tab "Descobreix", tap "Cerca activitats" and write what you look for ("Quina activitat busques?").
- Results show "N activitats trobades". Tap the clear icon ("Esborrar la cerca") to start again.
- "Descobreix" also shows "Per a tu" (based on your interests, empty until you do the interest test) and "Destacats".
- The "Persones" option to search people is not available yet.

## activity-details
Tap any activity card to open "Activitat". It shows "Sobre l'activitat", "Informació pràctica", "Publicat per" and the source ("Font: Agenda Cultural · Generalitat").
Not available yet: "Consultar informació original", "Xat de l'activitat" and "Veure perfil".

## sign-up
- To join an activity tap "M'hi apunto" on the card or on the activity page. The button changes to "Apuntat".
- To leave, tap "Apuntat", then "Desapunta'm" in the question "Vols desapuntar-te?". Tap "Mantén el pla" to stay.
- Activities you joined appear in the tab "Agenda".

## create-activity
- On "Inici" tap the "Crear activitat" button (+). The screen is "Crea una activitat".
- Fill: cover image (choose one of the provided images), Títol, Descripció, Categoria, "Data i hora d'inici", Municipi, tap the map to mark the place, and optionally "Indicacions (opcional)" and "Preu (opcional)".
- Tap "Publica". You are signed up automatically and the new activity opens.
- Uploading your own photo is not available yet. Editing or deleting an activity you created is not available yet.

## map
- On "Inici" use the switch "Mapa" / "Llista" to change between the map and the list. The list is shown first.
- Tap a marker to see a card with the button "Veure activitat". Tap a group of markers to zoom in.
- "Llegenda" explains the markers: official activities, "Publicada per un usuari" and "Publicada per una amistat".

## agenda
The tab "Agenda" shows a month calendar. Use "Mes anterior" and "Mes següent" to change month and tap a day to see your plans. If there are none it says "No tens cap pla per a aquest dia."

## notifications
- The tab "Avisos" lists your notices. Filters: "Tots", "Social", "Recordatoris" and "Recomanacions". Pull down to refresh.
- Tap a notice to mark it as read and open its activity.
- Friend requests show "Acceptar" and "Rebutjar".
- Reminders: in settings there is a "Recordatoris" switch, but scheduled reminders are not active yet.

## friends
- You can accept or reject a friend request from the tab "Avisos" ("Acceptar" or "Rebutjar").
- Your profile shows how many friends you have ("Amics"). On the map, activities published by a friend are marked "Publicada per una amistat".
- Not available yet: adding friends and seeing the plans of friends.
- Privacy: in settings, "Mostrar els meus plans" lets friends see what you sign up for.

## profile
- The tab "Perfil" shows your plans, friends, rating and level, and two lists: "Plans fets" and "Creats per mi".
- "Canviar foto" changes your avatar from the gallery. "Editar perfil" changes Nom, "Nom d'usuari" and Biografia. The municipality cannot be edited yet.
- "Compartir perfil" copies your @username.

## settings
Open the tab "Perfil" and tap the gear icon ("Configuració"). Sections:
- Preferences: "Idioma" and "Tema". Both are not available yet (the app is only in Catalan).
- Privacy: "Mostrar els meus plans" (works) and "Usuaris bloquejats" (not available yet).
- Notifications: "Recordatoris", "Activitat dels amics", "Recomanacions" and "Sol·licituds d'amistat". Only some switches are saved for now.
- Account: "Google Calendar" (not available yet), "Tornar a fer el test d'interessos", "Tancar sessió" and "Eliminar compte" (not available yet).

## interest-test
- Open it from "Configuració" > "Tornar a fer el test d'interessos".
- Swipe right if an activity interests you and left if not, or use the buttons "M'interessa" and "No m'interessa". Progress is shown as "N de M".
- At the end the app updates your interests and "Per a tu" in "Descobreix" improves. Liking does not sign you up.

## assistant
- Tap the "Assistent" button (robot icon) on "Inici".
- Write your question in "Escriu una consulta…" and tap "Enviar". You can ask for activities ("concerts a Girona aquest cap de setmana") or how to use the app.
- Under an answer about activities you see the activities as cards and chips with what the assistant understood ("Tipus", "Municipi", "Dates"). Tap a card to open it.
- Voice input is not available yet. If an error appears, tap "Tornar-ho a provar".
