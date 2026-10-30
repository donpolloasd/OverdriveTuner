# Firma digital con SignPath (gratis para código abierto)

Sin firma, Windows SmartScreen muestra «Windows protegió su PC / Editor desconocido» y Defender puede dar falsos positivos. SignPath Foundation firma gratis proyectos de código abierto. El certificado sale a nombre de **SignPath Foundation** (no al tuyo): eso es lo que va a decir «Editor» en Windows.

## Lo que ya está listo en el proyecto

- `.github/workflows/release.yml`: compila desde el código en GitHub Actions, firma el programa, lo mete firmado dentro del instalador y del parche, firma esos dos y crea un Release **en borrador** con `OverdriveTuner-Setup.exe`, `OverdriveTuner-Patch.exe`, `OverdriveTuner-Portable.exe` y `SHA256.txt`. Mientras no cargues los datos de SignPath, hace todo igual pero sin firmar (sirve para probarlo ya).
- Ficha de versión en los .exe (nombre del producto «OverdriveTuner» y versión): SignPath lo exige.
- `PRIVACIDAD.md`: la política de privacidad que piden (qué conexiones hace el programa y cómo desactivarlas).
- `LICENSE` (MIT, aprobada por OSI).

## Lo que tenés que hacer vos

1. **Subir todo a GitHub**, incluida la carpeta `.github` y `PRIVACIDAD.md`.
2. **Activar la verificación en dos pasos (2FA) en GitHub.** SignPath la exige a todo el equipo.
3. **Probar el flujo sin firma:** en GitHub, pestaña *Actions* → *Release* → *Run workflow*. Tiene que terminar en verde y dejar los .exe como artefacto. (Con un tag `v2.8.2` además crea el borrador del Release.)
4. **Publicar un Release** (aunque sea sin firmar). SignPath pide que el proyecto ya esté publicado en la forma que se va a firmar.
5. **Agregar al README** la sección de abajo («Firma de código»). Es obligatoria.
6. **Pedir el alta** en <https://signpath.org/apply>. Te van a pedir el repositorio, una descripción y quiénes aprueban cada firma. Tarda días o semanas y la aceptación queda a criterio de ellos.
7. **Cuando te aprueben**, en SignPath:
   - Creá el proyecto (slug, por ejemplo `OverdriveTuner`) y vinculale el *Trusted Build System* «GitHub.com» (viene predefinido).
   - En la configuración de artefactos, usá esta (firma todos los .exe que vengan en el zip que sube GitHub):
     ```xml
     <artifact-configuration xmlns="http://signpath.io/artifact-configuration/v1">
       <zip-file>
         <pe-file path="*.exe" product-name="OverdriveTuner">
           <authenticode-sign/>
         </pe-file>
       </zip-file>
     </artifact-configuration>
     ```
   - Creá un token de API para un usuario con permiso de envío (*submitter*).
8. **Cargar los datos en GitHub** (*Settings → Secrets and variables → Actions*):
   - Secreto `SIGNPATH_API_TOKEN`.
   - Variables `SIGNPATH_ORGANIZATION_ID`, `SIGNPATH_PROJECT_SLUG` y `SIGNPATH_POLICY_SLUG` (`release-signing`; para probar, `test-signing`).
9. **Cada versión nueva:** subís el tag (`git tag v2.8.3` y `git push origin v2.8.3`), aprobás las dos firmas en SignPath (cada firma se aprueba a mano), revisás el borrador del Release y lo publicás.

## Sección para el README (obligatoria)

```markdown
## Firma de código

Firma de código gratuita provista por [SignPath.io](https://about.signpath.io), certificado de [SignPath Foundation](https://signpath.org).

- Autores y revisores: [donpolloasd](https://github.com/donpolloasd)
- Aprobación de cada firma: [donpolloasd](https://github.com/donpolloasd)

Política de privacidad: este programa no envía datos a ningún lado salvo lo que se explica en [PRIVACIDAD.md](PRIVACIDAD.md).
```

## Ojo: lo que podría frenar la aprobación

Las condiciones de SignPath prohíben funciones que «eludan medidas de seguridad». OverdriveTuner tiene ajustes que **desactivan protecciones de Windows**, y alguno de estos podría ser un problema:

- «Desactivar VBS / Integridad de memoria (HVCI)» (Avanzado).
- «Desactivar mitigaciones Spectre/Meltdown» (Avanzado, ya oculto del catálogo).
- «Aplazar las actualizaciones de Windows hasta el año 3000», que también frena los parches de seguridad.

Todos piden doble confirmación, se revierten y ninguno entra en los perfiles. Explicalo así en la solicitud. Si igual te piden sacarlos, se pueden quitar del catálogo de la versión firmada (como se hizo en la 2.1 con otros ajustes) y dejarlos visibles solo para quien ya los tenía aplicados, para que pueda revertirlos.
