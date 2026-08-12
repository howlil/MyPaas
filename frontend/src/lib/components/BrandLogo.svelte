<script lang="ts">
	import markBlack from '../../assets/new-assets/logoonly_black.png';
	import markWhite from '../../assets/new-assets/logoonly_white.png';
	import wordmarkBlack from '../../assets/new-assets/logowithtext_black.png';
	import wordmarkWhite from '../../assets/new-assets/logowithtext_white.png';

	export let compact = false;
	export let className = '';
	export let imageClass = '';

	// Assets are tightly cropped, so sizing is driven optically: the wordmark is
	// width-led (~160px default) and the cube mark is height-led (32px default).
	$: sources = compact ? { light: markBlack, dark: markWhite } : { light: wordmarkBlack, dark: wordmarkWhite };
	$: sizing = compact ? 'h-8 w-auto' : 'w-40 h-auto';
</script>

<span class={`inline-flex min-w-0 items-center ${className}`.trim()}>
	<span class="sr-only">MyPaas</span>
	<img src={sources.light} alt="" aria-hidden="true" class={`${sizing} object-contain object-left dark:hidden ${imageClass}`.trim()} />
	<img src={sources.dark} alt="" aria-hidden="true" class={`${sizing} hidden object-contain object-left dark:block ${imageClass}`.trim()} />
</span>
